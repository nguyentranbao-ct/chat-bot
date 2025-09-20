# Chat-Bot Data Flow Details

This document provides detailed samples, schemas, and examples for each step in the data flow, including request/response data, prompts, and tool interactions.

## 1. Receive POST /api/v1/messages

**Endpoint:** `POST /api/v1/messages`

**Headers:**

- `x-project-uuid`: `16f38160-3afa-4707-b8cb-354d2cbf1590`
- `Service`: `chat-bot`
- `Content-Type`: `application/json`

**Request Body Schema:**

```json
{
  "channel_id": "string",
  "created_at": 1758015379886,
  "sender_id": "string",
  "message": "string",
  "attachment": {
    "id": "string",
    "type": "string",
    "data": {}
  },
  "type": "text",
  "metadata": {
    "llm": {
      "chat_mode": "seller_mode"
    }
  },
  "is_edited": false
}
```

**Sample Request:**

```json
{
  "channel_id": "30APeUsHYrde6THBN0ST7QgKcyA",
  "created_at": 1758015379886,
  "sender_id": "user123",
  "message": "Hi, I'm interested in buying this item. Can you tell me more?",
  "attachment": null,
  "type": "text",
  "metadata": {
    "llm": {
      "chat_mode": "seller_mode"
    }
  },
  "is_edited": false
}
```

## 2. Validate Request

**Validation Checks:**

- Headers present: `x-project-uuid`, `Service: chat-bot`
- Body fields: `channel_id`, `sender_id`, `message`, `metadata.llm.chat_mode`
- Chat mode exists in MongoDB `modes` collection

**Error Response (if invalid):**

```json
{
  "code": 400,
  "message": "Invalid request: missing metadata.llm.chat_mode"
}
```

## 3. Gather Data

**Channel Info Call:** `GET /api/v1/internal/channels/{channel_id}/user_channels`

**Headers:** Same as request.

**Response Schema:**

```json
{
  "code": 200,
  "data": [
    {
      "user_id": "string",
      "channel_id": "string",
      "name": "string",
      "avatar": "string",
      "item_name": "string",
      "item_image": "string",
      "item_price": "string",
      "is_hidden": false,
      "is_muted": false,
      "role": "string",
      "last_message": "string",
      "last_message_type": "string",
      "last_message_created_at": 1758015379886,
      "joined_at": 1753067499574,
      "channel_type": "string",
      "seller_id": "string",
      "list_id": "string",
      "category": "string",
      "metadata": {},
      "is_spam": false,
      "user_name": "string",
      "user_avatar": "string"
    }
  ]
}
```

**Sample Channel Info:**

```json
{
  "code": 200,
  "data": [
    {
      "user_id": "seller456",
      "channel_id": "30APeUsHYrde6THBN0ST7QgKcyA",
      "name": "Product Chat",
      "item_name": "Wireless Headphones",
      "item_price": "$99.99",
      "metadata": {}
    }
  ]
}
```

**Messages Call:** `GET /api/v1/internal/user_channel/{user_id}/{channel_id}/messages?limit=100&before_ts={current_ts}`

**Sample Messages Response:**

```json
{
  "code": 200,
  "data": [
    {
      "channel_id": "30APeUsHYrde6THBN0ST7QgKcyA",
      "created_at": 1758015379000,
      "sender_id": "user123",
      "message": "Hello, is this item still available?",
      "type": "text"
    },
    {
      "channel_id": "30APeUsHYrde6THBN0ST7QgKcyA",
      "created_at": 1758015379500,
      "sender_id": "seller456",
      "message": "Yes, it's available! What would you like to know?",
      "type": "text"
    }
  ]
}
```

## 4. Initialize Agent

**Mode Config Retrieval:** Using `metadata.llm.chat_mode`, retrieve the mode configuration from MongoDB `modes` collection. The config includes:

- prompt_template: Golang template string for the system prompt
- model: LLM model name (e.g., "gemini-2.5-flash")
- tools: List of tool names to enable (e.g., ["TriggerBuy", "ReplyMessage", "FetchMessages", "EndSession"])
- max_iterations: Maximum number of AI flow iterations (e.g., 10)
- max_prompt_tokens: Maximum input tokens per prompt (e.g., 32000)
- max_response_tokens: Maximum output tokens per response (e.g., 4000)

**LLM Config (from mode):**

- Model: From config.model
- Max Iterations: From config.max_iterations
- Token limit: From config.max_token_per_prompt per prompt, 150k quota

**Tools:**

- TriggerBuy: Logs buy intent to console and database
- ReplyMessage: Sends message via chat-api
- FetchMessages: Fetches more messages with pagination
- EndSession: Terminates conversation and updates session

**Agent Setup:** One-shot agent with system prompt, tools, and LLM.

## 5. Build Prompt

**System Prompt Template (from config):**

```
You are a helpful AI assistant acting as a {{.Role}} for {{.ItemName}} in channel {{.ChannelID}}.

**Role:** {{.RoleDescription}}
**Restrictions:**
- Only respond to product-related inquiries.
- Do not share personal information.
- Keep responses professional and concise.

**Available Tools:**

- TriggerBuy: Use when the user expresses clear intent to purchase the product (e.g., "I want to buy", "purchase", "yes buy it"). How to use: Respond with ToolName: TriggerBuy\nArguments: {"intent": "string describing the buy intent"}. This tool logs the buy intent to console and database.
- ReplyMessage: Use to send a reply message to the user as the seller. How to use: Respond with ToolName: ReplyMessage\nArguments: {"message": "string, the message content to send"}. This tool calls the chat-api to post the message to the channel.
- FetchMessages: Use to fetch additional conversation history for better context if the current history is insufficient. How to use: Respond with ToolName: FetchMessages\nArguments: {"limit": "int, number of messages to fetch (default 100)", "before_ts": "int64, unix millisecond timestamp to fetch messages before (optional)"}. This tool calls the chat-api to retrieve more messages.
- EndSession: Use to terminate the conversation when complete. How to use: Respond with ToolName: EndSession\nArguments: {}. This tool ends the AI flow and updates the session in database.

**How to Use Tools:**
- To use a tool, respond with: ToolName: [ToolName]\nArguments: [JSON]
- Arguments must be in exact JSON format, with correct symbols, quotes, and spacing.
- After tool execution, continue the conversation.

**Output Format:**
- For final answers: "Final Answer: [response]"
- Always end with a tool action or final answer.
```

**Note:** Go templates in chat mode settings can access channel fields (e.g., {{.ItemName}}, {{.ItemPrice}}) and metadata directly for dynamic customization.

**Rendered System Prompt:**

```
You are a helpful AI assistant acting as a Seller for Wireless Headphones in channel 30APeUsHYrde6THBN0ST7QgKcyA.

**Role:** Seller promoting and selling products.
**Restrictions:**
- Only respond to product-related inquiries.
- Do not share personal information.
- Keep responses professional and concise.

**Available Tools:**

- BuyTrigger: Use when the user expresses clear intent to purchase the product (e.g., "I want to buy", "purchase", "yes buy it"). How to use: Respond with
  - ToolName: TriggerBuy
    Arguments: {"intent": "string describing the buy intent"}. This tool logs the buy intent and sends a notification to the seller.
- SendReply: Use to send a reply message to the user as the seller.
  - ToolName: ReplyMessage
    Arguments: {"message": "string, the message content to send"}. This tool calls the chat-api to post the message to the channel.
- ListMessages: Use to fetch additional conversation history for better context if the current history is insufficient. How to use: Respond with
  - ToolName: ListMessages\nArguments: {"limit": "int, number of messages to fetch (default 100)", "before_ts": "int64, unix millisecond timestamp to fetch messages before (optional)"}. This tool calls the chat-api to retrieve more messages.

**How to Use Tools:**
- To use a tool, respond with: ToolName: [ToolName]\nArguments: [JSON]
- Arguments must be in exact JSON format, with correct symbols, quotes, and spacing.
- After tool execution, continue the conversation.

**Output Format:**
- For final answers: "Final Answer: [response]"
- Always end with a tool action or final answer.

**Conversation History:**
User: Hello, is this item still available?
AI: Yes, it's available! What would you like to know?
User: Hi, I'm interested in buying this item. Can you tell me more?
```

## 6. Run Agent Loop

**LLM Response (first iteration):**

```
Thought: The user is asking for more info about the headphones. I should provide details and offer to help with purchase.

ToolName: ReplyMessage
Arguments: {"message": "Our wireless headphones feature active noise cancellation, 30-hour battery life, and premium sound quality. Price is $99.99. Would you like to proceed with the purchase?"}
```

## 7. Tools Needed?

**Check:** Yes, "ToolName: ReplyMessage" detected.

## 8. Execute Tools

**ReplyMessage Tool Execution:**

- Call: `POST /api/v1/internal/messages`
- Body:

```json
{
  "channel_id": "30APeUsHYrde6THBN0ST7QgKcyA",
  "sender_id": "seller456",
  "message": "Our wireless headphones feature active noise cancellation, 30-hour battery life, and premium sound quality. Price is $99.99. Would you like to proceed with the purchase?",
  "type": "text"
}
```

- Response: `{"code": 200}`

**Tool Output:** "Reply sent successfully."

## 9. Append to Prompt

**Updated Prompt:**

```
You are a helpful AI assistant acting as a Seller for Wireless Headphones in channel 30APeUsHYrde6THBN0ST7QgKcyA.

**Role:** Seller promoting and selling products.
**Restrictions:**
- Only respond to product-related inquiries.
- Do not share personal information.
- Keep responses professional and concise.

**Available Tools:**

- TriggerBuy: Use when the user expresses clear intent to purchase the product (e.g., "I want to buy", "purchase", "yes buy it"). How to use: Respond with ToolName: TriggerBuy\nArguments: {"intent": "string describing the buy intent"}. This tool logs the buy intent to console and database.
- ReplyMessage: Use to send a reply message to the user as the seller. How to use: Respond with ToolName: ReplyMessage\nArguments: {"message": "string, the message content to send"}. This tool calls the chat-api to post the message to the channel.
- FetchMessages: Use to fetch additional conversation history for better context if the current history is insufficient. How to use: Respond with ToolName: FetchMessages\nArguments: {"limit": "int, number of messages to fetch (default 100)", "before_ts": "int64, unix millisecond timestamp to fetch messages before (optional)"}. This tool calls the chat-api to retrieve more messages.
- EndSession: Use to terminate the conversation when complete. How to use: Respond with ToolName: EndSession\nArguments: {}. This tool ends the AI flow and updates the session in database.

**How to Use Tools:**
- To use a tool, respond with: ToolName: [ToolName]\nArguments: [JSON]
- Arguments must be in exact JSON format, with correct symbols, quotes, and spacing.
- After tool execution, continue the conversation.

**Output Format:**
- For final answers: "Final Answer: [response]"
- Always end with a tool action or final answer.

**Conversation History:**
User: Hello, is this item still available?
AI: Yes, it's available! What would you like to know?
User: Hi, I'm interested in buying this item. Can you tell me more?

**Previous Tool Use:**
ToolName: ReplyMessage
Arguments: {"message": "Our wireless headphones feature active noise cancellation, 30-hour battery life, and premium sound quality. Price is $99.99. Would you like to proceed with the purchase?"}

**Tool Response:**
Reply sent successfully.
```

## 10. Continue Loop or End

**Loop Continuation:** The agent loop continues until the LLM decides to end it. The loop ends condition:

1. **End without Tool Use:** LLM responds without specifying a tool, either with "Final Answer: [response]" or without it (ending based on context or max iterations).
2. **End with Tool Use:** LLM specifies a tool, either with "Final Answer: [response]" in the same response (execute tool, then end) or as a final action (e.g., TriggerBuy Tool; execute and end).

**Example Second Iteration:**

**Input to Agent (next user message):**

```
{"input": "Yes, buy it!"}
```

**LLM Response (second iteration):**

```
Thought: The user has expressed clear intent to buy. I should trigger the buy process.

ToolName: TriggerBuy
Arguments: {"intent": "Yes, buy it!"}
```

**Tools Needed?**

**Check:** Yes, "ToolName: TriggerBuy" detected.

**Execute Tools**

**TriggerBuy Tool Execution:**

- Log: `Buy intent detected: Yes, buy it!`
- Output: "Logged buy intent."

**Loop Ends:** After TriggerBuy Tool execution, the loop ends as this is the final action for purchase intent.
