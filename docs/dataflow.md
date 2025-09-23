# Data Flow and Message Processing

## Table of Contents

- [Overview](#overview)
- [Message Input Sources](#message-input-sources)
- [HTTP API Message Flow](#http-api-message-flow)
- [Kafka Consumer Message Flow](#kafka-consumer-message-flow)
- [Partner Abstraction Flow](#partner-abstraction-flow)
- [LLM Processing Pipeline](#llm-processing-pipeline)
- [Tool Execution Flows](#tool-execution-flows)
- [Error Handling & Recovery](#error-handling--recovery)

## Overview

The Chat-Bot system processes messages through two primary channels with built-in **loop prevention** and **partner abstraction**. Both flows converge at the LLM processing pipeline, ensuring consistent conversation handling regardless of input source.

### Key Flow Characteristics

- **Dual Input Processing**: HTTP API (synchronous) and Kafka consumer (asynchronous)
- **Partner Abstraction**: Unified partner detection and routing
- **Loop Prevention**: Internal user identification prevents infinite loops
- **Tool-Based AI**: Iterative LLM calls with tool execution
- **Session Management**: Complete conversation lifecycle tracking

## Message Input Sources

```mermaid
graph TB
    subgraph "External Sources"
        ChototAPI[Chotot Chat-API]
        FuturePartners[Future Partners<br/>Facebook, WhatsApp, etc.]
        TestClients[Test Clients<br/>Postman, Frontend]
    end

    subgraph "Input Channels"
        HTTPEndpoint[HTTP POST /api/v1/messages]
        KafkaConsumer[Kafka Consumer<br/>chat.event.messages]
    end

    subgraph "Chat-Bot Service"
        MessageProcessor[Message Processor]
        PartnerRegistry[Partner Registry]
        LoopPrevention[Internal User Check]
        LLMPipeline[LLM Processing Pipeline]
    end

    ChototAPI --> HTTPEndpoint
    ChototAPI --> KafkaConsumer
    TestClients --> HTTPEndpoint
    FuturePartners -.-> HTTPEndpoint
    FuturePartners -.-> KafkaConsumer

    HTTPEndpoint --> MessageProcessor
    KafkaConsumer --> MessageProcessor
    MessageProcessor --> PartnerRegistry
    MessageProcessor --> LoopPrevention
    LoopPrevention --> LLMPipeline
```

## HTTP API Message Flow

### Flowchart - HTTP Message Processing

```mermaid
flowchart TD
    A[HTTP POST /api/v1/messages] --> B{Validate Headers}
    B -->|Missing Headers| C[Return 400 Error<br/>x-project-uuid, Service required]
    B -->|Valid| D[Extract Message Data]

    D --> E[Partner Detection<br/>from Source]
    E --> F{Partner Registered?}
    F -->|No| G[Return 400 Error<br/>Unknown Partner]
    F -->|Yes| H[Internal User Check]

    H --> I{Is Internal User?}
    I -->|Yes| J[Return 200 OK<br/>Skip Processing]
    I -->|No| K[Continue Processing]

    K --> L[Gather Room Information<br/>via Partner API]
    L --> M{Room Data Valid?}
    M -->|No| N[Return 404 Error<br/>Room Not Found]
    M -->|Yes| O[Fetch Message History<br/>via Partner API]

    O --> P[Build LLM Prompt<br/>System + Context + History]
    P --> Q[Initialize AI Agent<br/>with Chat Mode Config]
    Q --> R[Start LLM Processing Loop]

    R --> S{Tools Needed?}
    S -->|Yes| T[Execute Tools<br/>ReplyMessage, PurchaseIntent, etc.]
    T --> U[Append Tool Results to Prompt]
    U --> V{Max Iterations Reached?}
    V -->|No| R
    V -->|Yes| W[Force End Session]
    S -->|No| X[End Processing Loop]

    W --> Y[Log Session Completion]
    X --> Y
    Y --> Z[Return 200 OK<br/>Processing Complete]

    style A fill:#e1f5fe
    style C fill:#ffebee
    style G fill:#ffebee
    style J fill:#e8f5e8
    style N fill:#ffebee
    style Z fill:#e8f5e8
```

### Sequence Diagram - HTTP Message Processing

```mermaid
sequenceDiagram
    participant Client as Chotot Chat-API
    participant HTTP as HTTP Server
    participant Partner as Partner Registry
    participant UserRepo as User Repository
    participant Repo as Room Repository
    participant LLM as LLM Processor
    participant Tools as Tool Manager
    participant ExtAPI as External APIs

    Client->>HTTP: POST /api/v1/messages
    Note over Client,HTTP: Headers: x-project-uuid, Service: chat-bot

    HTTP->>HTTP: Validate Request Headers
    HTTP->>Partner: Detect Partner from Source
    Partner-->>HTTP: Partner Interface

    HTTP->>UserRepo: Check if Sender is Internal User
    UserRepo-->>HTTP: External User (Continue)

    Note over HTTP: Skip processing if internal user detected
    HTTP->>Repo: Get Room Information
    Repo->>ExtAPI: Fetch Room Data via Partner API
    ExtAPI-->>Repo: Room Details + Participants
    Repo-->>HTTP: Room Context

    HTTP->>Repo: Get Message History
    Repo->>ExtAPI: Fetch Messages via Partner API
    ExtAPI-->>Repo: Message History
    Repo-->>HTTP: Historical Messages

    HTTP->>LLM: Build Prompt & Start Processing
    Note over LLM: System + Context + History

    loop AI Processing Loop
        LLM->>LLM: Generate Response
        LLM->>Tools: Parse Required Tools

        alt Tools Needed
            Tools->>ExtAPI: Execute ReplyMessage
            ExtAPI-->>Tools: Message Sent
            Tools->>Repo: Log Purchase Intent
            Repo-->>Tools: Intent Recorded
            Tools-->>LLM: Tool Results
        else No Tools
            LLM->>LLM: End Processing
        end
    end

    LLM->>Repo: Log Session Activity
    LLM-->>HTTP: Processing Complete
    HTTP-->>Client: 200 OK
```

## Kafka Consumer Message Flow

### Flowchart - Kafka Message Processing

```mermaid
flowchart TD
    A[Kafka Message Received<br/>chat.event.messages] --> B[Parse Kafka Message<br/>Extract Data from JSON]
    B --> C{Valid Message Format?}
    C -->|Invalid JSON| D[Log Error & Skip<br/>Continue to Next Message]
    C -->|Valid| E[Extract Channel & Metadata]

    E --> F{Channel in Whitelist?}
    F -->|No| G[Skip Message<br/>Not in Configured Channels]
    F -->|Yes| H[Extract LLM Chat Mode<br/>from Metadata]

    H --> I{Chat Mode Valid?}
    I -->|Invalid| J[Log Warning & Skip<br/>No Valid Chat Mode]
    I -->|Valid| K[Partner Detection<br/>from Message Source]

    K --> L{Partner Registered?}
    L -->|No| M[Log Error & Skip<br/>Unknown Partner]
    L -->|Yes| N[Internal User Check]

    N --> O{Is Internal User?}
    O -->|Yes| P[Skip Processing<br/>Prevent Infinite Loop]
    O -->|No| Q[Continue Processing]

    Q --> R[Transform to Internal Format<br/>Normalize Message Structure]
    R --> S[Continue with Standard Flow<br/>Same as HTTP from here]

    S --> T[Gather Room Information]
    T --> U[Fetch Message History]
    U --> V[Build LLM Prompt]
    V --> W[Execute AI Processing Loop]
    W --> X[Log Completion]

    style A fill:#fff3e0
    style D fill:#ffebee
    style G fill:#fff8e1
    style J fill:#fff8e1
    style M fill:#ffebee
    style P fill:#fff8e1
    style X fill:#e8f5e8
```

### Sequence Diagram - Kafka Message Processing

```mermaid
sequenceDiagram
    participant Kafka as Kafka Broker
    participant Consumer as Kafka Consumer
    participant Whitelist as Channel Whitelist
    participant Partner as Partner Registry
    participant Dedup as Message Deduplication
    participant Processor as Message Processor
    participant LLM as LLM Pipeline

    Kafka->>Consumer: Message from chat.event.messages
    Consumer->>Consumer: Parse JSON Message

    Consumer->>Whitelist: Check Channel Whitelist
    alt Channel Not Whitelisted
        Whitelist-->>Consumer: Skip Message
        Consumer->>Consumer: Continue to Next Message
    else Channel Whitelisted
        Whitelist-->>Consumer: Process Message

        Consumer->>Consumer: Extract LLM Metadata
        Consumer->>Partner: Detect Partner from Source
        Partner-->>Consumer: Partner Interface

        Consumer->>Dedup: Check Message Deduplication
        alt Already Processed
            Dedup-->>Consumer: Duplicate Found
            Consumer->>Consumer: Skip Processing
        else New Message
            Dedup-->>Consumer: Not Processed
            Consumer->>Dedup: Record Processing

            Consumer->>Processor: Transform & Forward Message
            Note over Processor: Same flow as HTTP from here
            Processor->>LLM: Start LLM Processing

            LLM->>LLM: Execute AI Pipeline
            LLM-->>Processor: Processing Complete
            Processor-->>Consumer: Success
        end
    end

    Consumer->>Kafka: Commit Message Offset
```

## Partner Abstraction Flow

### Partner Detection and Routing

```mermaid
flowchart TD
    A[Incoming Message<br/>Any Source] --> B[Extract Source Information<br/>Partner Name, Room ID, Message ID]
    B --> C[Partner Registry Lookup]

    C --> D{Partner Exists?}
    D -->|No| E[Return Error<br/>Unknown Partner: facebook, whatsapp]
    D -->|Yes| F[Get Partner Interface<br/>Chotot, Future Partners]

    F --> G[Partner-Specific Operations]

    subgraph "Partner Operations"
        H[Get Room Information<br/>via Partner API]
        I[Fetch Message History<br/>via Partner API]
        J[Send Reply Message<br/>via Partner API]
        K[Get User Products<br/>via Partner API]
    end

    G --> H
    G --> I
    G --> J
    G --> K

    H --> L[Standardized Room Data]
    I --> M[Standardized Message History]
    J --> N[Message Delivery Confirmation]
    K --> O[Product List Response]

    style A fill:#e3f2fd
    style E fill:#ffebee
    style L fill:#e8f5e8
    style M fill:#e8f5e8
    style N fill:#e8f5e8
    style O fill:#e8f5e8
```

### Current and Future Partner Support

```mermaid
graph LR
    subgraph "Partner Registry"
        Registry[Partner Registry<br/>Interface]
    end

    subgraph "Current Implementation"
        Chotot[Chotot Partner<br/>✅ Implemented]
        ChototChat[Chotot Chat-API]
        ChototProduct[Chotot Product API]
    end

    subgraph "Future Partners"
        Facebook[Facebook Partner<br/>🔄 Framework Ready]
        WhatsApp[WhatsApp Partner<br/>🔄 Framework Ready]
        Telegram[Telegram Partner<br/>🔄 Framework Ready]

        FacebookAPI[Facebook Messenger API]
        WhatsAppAPI[WhatsApp Business API]
        TelegramAPI[Telegram Bot API]
    end

    Registry --> Chotot
    Registry -.-> Facebook
    Registry -.-> WhatsApp
    Registry -.-> Telegram

    Chotot --> ChototChat
    Chotot --> ChototProduct

    Facebook -.-> FacebookAPI
    WhatsApp -.-> WhatsAppAPI
    Telegram -.-> TelegramAPI

    style Chotot fill:#4caf50,color:#fff
    style Facebook fill:#2196f3,color:#fff
    style WhatsApp fill:#4caf50,color:#fff
    style Telegram fill:#2196f3,color:#fff
```

## LLM Processing Pipeline

### AI Agent Execution Flow

```mermaid
flowchart TD
    A[Start LLM Processing] --> B[Load Chat Mode Configuration<br/>from MongoDB]
    B --> C[Build System Prompt<br/>Template + Context Variables]

    C --> D[Append Message History<br/>Conversation Context]
    D --> E[Initialize Genkit Agent<br/>Model + Tools + Config]

    E --> F[Execute LLM Call<br/>Send Prompt to Model]
    F --> G[Parse LLM Response<br/>Extract Tool Calls]

    G --> H{Tools Required?}
    H -->|No| I[End Processing<br/>Save Session State]
    H -->|Yes| J[Execute Tool Calls<br/>Parallel Execution]

    J --> K[Append Tool Results<br/>to Conversation Prompt]
    K --> L{Max Iterations Reached?}
    L -->|Yes| M[Force End Session<br/>Log Max Iterations Warning]
    L -->|No| N{Continue Processing?}
    N -->|Yes| F
    N -->|No| I

    M --> I
    I --> O[Log Session Activity<br/>Final State & Metrics]

    style A fill:#e1f5fe
    style I fill:#e8f5e8
    style M fill:#fff3e0
    style O fill:#e8f5e8
```

### Chat Mode Configuration Flow

```mermaid
sequenceDiagram
    participant Processor as Message Processor
    participant ModeRepo as Chat Mode Repository
    participant Template as Template Engine
    participant Genkit as Firebase Genkit
    participant Model as LLM Provider

    Processor->>ModeRepo: Get Chat Mode by Name
    ModeRepo-->>Processor: Chat Mode Config

    Note over Processor: Chat Mode Contains:
    Note over Processor: - Prompt Template
    Note over Processor: - Model Selection
    Note over Processor: - Tool Configuration
    Note over Processor: - Token Limits

    Processor->>Template: Render Prompt Template
    Note over Template: Variables: Room Context,<br/>Item Info, User Data
    Template-->>Processor: Rendered System Prompt

    Processor->>Genkit: Initialize Agent
    Note over Genkit: Model: gemini-1.5-pro<br/>Tools: [ReplyMessage, PurchaseIntent]<br/>Max Tokens: 4000

    loop AI Processing Loop
        Processor->>Genkit: Execute Agent Call
        Genkit->>Model: Send Prompt + Tool Definitions
        Model-->>Genkit: Response + Tool Calls
        Genkit-->>Processor: Parsed Response & Tools

        alt Tools Present
            Processor->>Processor: Execute Tools
            Processor->>Genkit: Append Tool Results
        else No Tools
            Processor->>Processor: End Processing
        end
    end
```

## Tool Execution Flows

### Available Tools and Their Functions

```mermaid
graph TB
    subgraph "LLM Tool Ecosystem"
        Agent[LLM Agent]

        subgraph "Communication Tools"
            ReplyMessage[ReplyMessage<br/>Send responses to users]
            FetchMessages[FetchMessages<br/>Get conversation history]
        end

        subgraph "Business Logic Tools"
            PurchaseIntent[PurchaseIntent<br/>Log buy signals]
            ListProducts[ListProducts<br/>Search user products]
        end

        subgraph "Session Management Tools"
            EndSession[EndSession<br/>Terminate conversation]
        end

        subgraph "External Services"
            PartnerAPI[Partner Chat APIs]
            ProductAPI[Partner Product APIs]
            Database[MongoDB Collections]
        end
    end

    Agent --> ReplyMessage
    Agent --> FetchMessages
    Agent --> PurchaseIntent
    Agent --> ListProducts
    Agent --> EndSession

    ReplyMessage --> PartnerAPI
    FetchMessages --> PartnerAPI
    ListProducts --> ProductAPI
    PurchaseIntent --> Database
    EndSession --> Database

    style Agent fill:#e1f5fe
    style ReplyMessage fill:#e8f5e8
    style FetchMessages fill:#e8f5e8
    style PurchaseIntent fill:#fff3e0
    style ListProducts fill:#e8f5e8
    style EndSession fill:#ffecb3
```

### Tool Execution Sequence

```mermaid
sequenceDiagram
    participant LLM as LLM Agent
    participant Tools as Tool Manager
    participant Partner as Partner API
    participant DB as MongoDB
    participant Session as Session Manager

    Note over LLM: AI decides tools needed based on conversation

    LLM->>Tools: Execute ReplyMessage
    Tools->>Partner: Send message via partner API
    Partner-->>Tools: Message delivery confirmation
    Tools->>DB: Log chat activity
    Tools-->>LLM: Tool execution result

    LLM->>Tools: Execute PurchaseIntent
    Note over Tools: User showed strong buying interest
    Tools->>DB: Create purchase intent record
    Tools->>Session: Update session analytics
    Tools-->>LLM: Intent logged successfully

    LLM->>Tools: Execute ListProducts
    Tools->>Partner: Search user's product listings
    Partner-->>Tools: Product list response
    Tools-->>LLM: Formatted product information

    alt End Conversation
        LLM->>Tools: Execute EndSession
        Tools->>Session: Mark session as ended
        Tools->>DB: Final activity log
        Tools-->>LLM: Session terminated
    else Continue Conversation
        LLM->>LLM: Continue processing
    end
```

### ReplyMessage Tool Flow

```mermaid
flowchart TD
    A[ReplyMessage Tool Called] --> B[Extract Message Content<br/>from LLM Response]
    B --> C[Get Partner Interface<br/>from Room Context]
    C --> D[Format Message for Partner<br/>Platform-Specific Formatting]

    D --> E{Partner API Call}
    E -->|Success| F[Message Sent Successfully]
    E -->|Failed| G[Retry with Exponential Backoff]

    G --> H{Retry Successful?}
    H -->|Yes| F
    H -->|No| I[Log Error & Mark as Failed]

    F --> J[Log Chat Activity<br/>Action: reply_message]
    I --> K[Log Error Activity<br/>Include Error Details]

    J --> L[Return Success to LLM]
    K --> M[Return Error to LLM]

    style A fill:#e1f5fe
    style F fill:#e8f5e8
    style I fill:#ffebee
    style L fill:#e8f5e8
    style M fill:#ffebee
```

### PurchaseIntent Tool Flow

```mermaid
flowchart TD
    A[PurchaseIntent Tool Called] --> B[Extract Intent Parameters<br/>from LLM Analysis]
    B --> C[Parse Intent Data<br/>Item, Price, Confidence]

    C --> D[Validate Intent Strength<br/>Percentage Threshold Check]
    D --> E{Strong Intent?}
    E -->|< 50%| F[Log as Low Intent<br/>Continue Monitoring]
    E -->|≥ 50%| G[Create Purchase Intent Record]

    G --> H[Store in MongoDB<br/>purchase_intents Collection]
    H --> I[Update Session Analytics<br/>Mark High-Value Session]

    F --> J[Log Chat Activity<br/>Intent: interest]
    I --> K[Log Chat Activity<br/>Intent: strong_buy_signal]

    J --> L[Return to LLM<br/>Continue Conversation]
    K --> M[Return to LLM<br/>Consider Follow-up Actions]

    style A fill:#e1f5fe
    style E fill:#fff3e0
    style F fill:#fff8e1
    style G fill:#e8f5e8
    style L fill:#e1f5fe
    style M fill:#e8f5e8
```

## Error Handling & Recovery

### Error Classification and Recovery Strategies

```mermaid
flowchart TD
    A[Error Occurs in Pipeline] --> B{Error Type Classification}

    B -->|Validation Error| C[Client Error 4xx<br/>Bad Request, Missing Headers]
    B -->|Partner API Error| D[External Service Error<br/>Chat-API, Product API Down]
    B -->|Database Error| E[Infrastructure Error<br/>MongoDB Connection Issues]
    B -->|LLM Error| F[AI Service Error<br/>Genkit, Model Provider Issues]

    C --> G[Return Error to Client<br/>No Retry]

    D --> H{Transient Error?}
    H -->|Yes| I[Exponential Backoff Retry<br/>Max 3 Attempts]
    H -->|No| J[Log Error & Fail Gracefully<br/>Continue Processing Other Messages]

    E --> K[Circuit Breaker Pattern<br/>Fallback to Cached Data]
    K --> L[Alert Operations Team<br/>Critical Infrastructure Issue]

    F --> M{Model Available?}
    M -->|Provider Down| N[Switch to Backup Provider<br/>Fallback Model]
    M -->|Rate Limited| O[Implement Backpressure<br/>Queue Messages]

    I --> P{Retry Successful?}
    P -->|Yes| Q[Continue Normal Processing]
    P -->|No| J

    N --> Q
    O --> Q

    style C fill:#ffebee
    style G fill:#ffebee
    style J fill:#fff3e0
    style L fill:#ff5722,color:#fff
    style Q fill:#e8f5e8
```

### Message Deduplication Prevention

```mermaid
sequenceDiagram
    participant Source as Message Source
    participant Dedup as Deduplication Service
    participant DB as MongoDB
    participant Processor as Message Processor

    Source->>Dedup: New Message with External ID
    Dedup->>DB: Check message_deduplication collection

    alt Message Already Processed
        DB-->>Dedup: External ID exists
        Dedup-->>Source: Skip - Already processed
        Note over Dedup: Prevents infinite loops
    else New Message
        DB-->>Dedup: External ID not found
        Dedup->>DB: Record External ID + timestamp
        Dedup->>Processor: Forward for processing

        Processor->>Processor: Execute LLM Pipeline
        Processor-->>Dedup: Processing complete

        Note over DB: TTL: 30 days for cleanup
    end
```

The data flow architecture ensures robust, scalable message processing with comprehensive error handling and partner abstraction, supporting both current operations and future platform expansion.