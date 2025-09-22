# Chat Bot API Documentation

## Receive New Message Event

```
POST /api/v1/messages
```

**Headers:**

- `x-project-uuid`: Project UUID (format: standard UUID)
- `Service`: chat-bot

**Request Body:**

```json
{
  "room_id": "string",
  "created_at": 1758015379886,
  "sender_id": "string",
  "message": "string",
  "attachment": {
    "id": "string",
    "type": "string",
    "data": {}
  },
  "type": "string",
  "metadata": {
    "llm": {
      "chat_mode": "seller_mode"
    }
  },
  "is_edited": false
}
```

**Example:**

```bash
curl --location 'http://chat-bot.chat/api/v1/messages' \
--header 'x-project-uuid: 16f38160-3afa-4707-b8cb-354d2cbf1590' \
--header 'Service: chat-bot' \
--header 'Content-Type: application/json' \
--data '{
  "room_id": "string",
  "created_at": 1758015379886,
  "sender_id": "string",
  "message": "string",
  "attachment": {
    "id": "string",
    "type": "string",
    "data": {}
  },
  "type": "string",
  "metadata": {
    "llm": {
      "chat_mode": "seller_mode"
    }
  },
  "is_edited": false
}'
```

**Response (200):**

```json
{
  "code": 200,
  "message": "Message received successfully"
}
```

**Error Responses:**

- **400 Bad Request**: Missing required fields or invalid metadata
- **401 Unauthorized**: Invalid project UUID or service header
- **500 Internal Server Error**: Database or LLM service error
