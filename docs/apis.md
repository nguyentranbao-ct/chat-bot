# API Reference Documentation

## Table of Contents

- [Overview](#overview)
- [Authentication](#authentication)
- [Message Processing APIs](#message-processing-apis)
- [User Management APIs](#user-management-apis)
- [Authentication APIs](#authentication-apis)
- [Chat Management APIs](#chat-management-apis)
- [Profile Management APIs](#profile-management-apis)
- [Error Handling](#error-handling)
- [Request/Response Examples](#requestresponse-examples)

## Overview

The Chat-Bot API provides comprehensive endpoints for message processing, user management, authentication, and chat operations. The API follows RESTful conventions with JSON request/response formats.

### Base URL
```
http://localhost:8080/api/v1
```

### Common Headers
```http
Content-Type: application/json
Authorization: Bearer <jwt_token>  # For authenticated endpoints
```

### Status Codes
- `200 OK` - Request successful
- `201 Created` - Resource created successfully
- `400 Bad Request` - Invalid request data
- `401 Unauthorized` - Authentication required or invalid
- `404 Not Found` - Resource not found
- `500 Internal Server Error` - Server error

## Authentication

### JWT Token Authentication
Most endpoints require JWT token authentication via the `Authorization` header:

```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

### Public Endpoints (No Authentication Required)
- `GET /health`
- `POST /auth/login`
- `POST /users` (User creation)
- `POST /internal/messages` (Internal message processing)

## Message Processing APIs

### Process Message via HTTP

Endpoint for external partners to send messages for AI processing.

```http
POST /api/v1/messages
```

**Required Headers:**
```http
x-project-uuid: <project_identifier>
Service: chat-bot
Content-Type: application/json
```

**Request Body:**
```json
{
  "room_id": "chotot_room_12345",
  "sender_id": "user_789",
  "content": "Is this iPhone still available?",
  "created_at": 1640995200000,
  "metadata": {
    "llm": {
      "chat_mode": "product_sales_assistant"
    }
  },
  "partner": {
    "name": "chotot",
    "room_id": "chotot_room_12345",
    "msg_id": "msg_456"
  }
}
```

**Response:**
```json
{
  "status": "success",
  "message": "Message processed successfully"
}
```

**Error Responses:**
```json
{
  "error": "Missing required header: x-project-uuid"
}
```

### Internal Message Processing

Internal endpoint for system-generated messages.

```http
POST /api/v1/internal/messages
```

**Request Body:**
```json
{
  "room_id": "507f1f77bcf86cd799439011",
  "sender_id": "507f1f77bcf86cd799439012",
  "content": "System notification message",
  "skip_partner": true
}
```

**Response:**
```json
{
  "message": "Message sent successfully",
  "id": "507f1f77bcf86cd799439013"
}
```

## User Management APIs

### Create User

```http
POST /api/v1/users
```

**Request Body:**
```json
{
  "name": "John Merchant",
  "email": "john@example.com"
}
```

**Response (201 Created):**
```json
{
  "id": "507f1f77bcf86cd799439011",
  "name": "John Merchant",
  "email": "john@example.com",
  "is_active": true,
  "is_internal": false,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

### Get User

```http
GET /api/v1/users/{id}
```

**Response:**
```json
{
  "id": "507f1f77bcf86cd799439011",
  "name": "John Merchant",
  "email": "john@example.com",
  "is_active": true,
  "is_internal": false,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

### Update User

```http
PUT /api/v1/users/{id}
```

**Request Body:**
```json
{
  "name": "John Updated Merchant",
  "email": "john.updated@example.com"
}
```

**Response:**
```json
{
  "id": "507f1f77bcf86cd799439011",
  "name": "John Updated Merchant",
  "email": "john.updated@example.com",
  "is_active": true,
  "is_internal": false,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T10:30:00Z"
}
```

### Delete User

```http
DELETE /api/v1/users/{id}
```

**Response:**
```json
{
  "status": "success",
  "message": "user deleted successfully"
}
```

## User Attributes APIs

### Set User Attribute

```http
POST /api/v1/users/{id}/attributes
```

**Request Body:**
```json
{
  "key": "chotot_id",
  "value": "merchant123",
  "tags": ["chotot", "primary"]
}
```

**Response:**
```json
{
  "status": "success",
  "message": "user attribute set successfully"
}
```

### Get All User Attributes

```http
GET /api/v1/users/{id}/attributes
```

**Response:**
```json
[
  {
    "id": "507f1f77bcf86cd799439014",
    "user_id": "507f1f77bcf86cd799439011",
    "key": "chotot_id",
    "value": "merchant123",
    "tags": ["chotot", "primary"],
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  },
  {
    "id": "507f1f77bcf86cd799439015",
    "user_id": "507f1f77bcf86cd799439011",
    "key": "whatsapp_phone_number_id",
    "value": "+1234567890",
    "tags": ["whatsapp", "phone"],
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
]
```

### Get Specific User Attribute

```http
GET /api/v1/users/{id}/attributes/{key}
```

**Response:**
```json
{
  "id": "507f1f77bcf86cd799439014",
  "user_id": "507f1f77bcf86cd799439011",
  "key": "chotot_id",
  "value": "merchant123",
  "tags": ["chotot", "primary"],
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

### Remove User Attribute

```http
DELETE /api/v1/users/{id}/attributes/{key}
```

**Response:**
```json
{
  "status": "success",
  "message": "user attribute removed successfully"
}
```

## Authentication APIs

### Login

```http
POST /api/v1/auth/login
```

**Request Body:**
```json
{
  "email": "john@example.com"
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "507f1f77bcf86cd799439011",
    "name": "John Merchant",
    "email": "john@example.com",
    "is_active": true,
    "is_internal": false,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  },
  "expires_at": "2024-01-08T00:00:00Z",
  "has_partner_attributes": true
}
```

### Get Profile

**🔒 Authentication Required**

```http
GET /api/v1/auth/me
```

**Response:**
```json
{
  "id": "507f1f77bcf86cd799439011",
  "name": "John Merchant",
  "email": "john@example.com",
  "is_active": true,
  "is_internal": false,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

### Update Profile

**🔒 Authentication Required**

```http
PUT /api/v1/auth/profile
```

**Request Body:**
```json
{
  "name": "John Updated Merchant"
}
```

**Response:**
```json
{
  "id": "507f1f77bcf86cd799439011",
  "name": "John Updated Merchant",
  "email": "john@example.com",
  "is_active": true,
  "is_internal": false,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T10:30:00Z"
}
```

### Logout

**🔒 Authentication Required**

```http
POST /api/v1/auth/logout
```

**Response:**
```json
{
  "status": "success",
  "message": "logged out successfully"
}
```

## Chat Management APIs

### Get User Rooms

**🔒 Authentication Required**

```http
GET /api/v1/chat/rooms
```

**Response:**
```json
[
  {
    "id": "507f1f77bcf86cd799439020",
    "name": "iPhone 12 Pro Discussion",
    "item_name": "iPhone 12 Pro",
    "item_price": "$899",
    "context": "Selling iPhone 12 Pro 256GB",
    "created_at": "2024-01-01T09:00:00Z",
    "updated_at": "2024-01-01T10:35:00Z",
    "last_read_at": "2024-01-01T10:30:00Z",
    "last_message_at": "2024-01-01T10:35:00Z",
    "last_message_content": "Is this still available?",
    "is_archived": false,
    "unread_count": 3
  }
]
```

### Get Room Members

**🔒 Authentication Required**

```http
GET /api/v1/chat/rooms/{id}/members
```

**Response:**
```json
[
  {
    "id": "507f1f77bcf86cd799439021",
    "user_id": "507f1f77bcf86cd799439011",
    "role": "merchant",
    "joined_at": "2024-01-01T09:00:00Z"
  },
  {
    "id": "507f1f77bcf86cd799439022",
    "user_id": "507f1f77bcf86cd799439012",
    "role": "buyer",
    "joined_at": "2024-01-01T09:05:00Z"
  }
]
```

### Send Message

**🔒 Authentication Required**

```http
POST /api/v1/chat/rooms/{id}/messages
```

**Request Body:**
```json
{
  "content": "Hello! Yes, the iPhone is still available.",
  "message_type": "text",
  "metadata": {
    "reply_to": "previous_message_id"
  }
}
```

**Response (201 Created):**
```json
{
  "id": "507f1f77bcf86cd799439025",
  "room_id": "507f1f77bcf86cd799439020",
  "sender_id": "507f1f77bcf86cd799439011",
  "content": "Hello! Yes, the iPhone is still available.",
  "metadata": {
    "reply_to": "previous_message_id",
    "source": "user"
  },
  "created_at": "2024-01-01T10:40:00Z",
  "updated_at": "2024-01-01T10:40:00Z"
}
```

### Get Room Messages

**🔒 Authentication Required**

```http
GET /api/v1/chat/rooms/{id}/messages?limit=50&before=507f1f77bcf86cd799439025
```

**Query Parameters:**
- `limit` (optional): Number of messages to retrieve (default: 50)
- `before` (optional): Get messages before this message ID (for pagination)

**Response:**
```json
[
  {
    "id": "507f1f77bcf86cd799439025",
    "room_id": "507f1f77bcf86cd799439020",
    "sender_id": "507f1f77bcf86cd799439011",
    "content": "Hello! Yes, the iPhone is still available.",
    "created_at": "2024-01-01T10:40:00Z"
  },
  {
    "id": "507f1f77bcf86cd799439024",
    "room_id": "507f1f77bcf86cd799439020",
    "sender_id": "507f1f77bcf86cd799439012",
    "content": "Is this still available?",
    "created_at": "2024-01-01T10:35:00Z"
  }
]
```

### Get Room Events

**🔒 Authentication Required**

```http
GET /api/v1/chat/rooms/{id}/events?since=1640995200
```

**Query Parameters:**
- `since` (optional): Unix timestamp to get events since (default: last 24 hours)

**Response:**
```json
[
  {
    "id": "507f1f77bcf86cd799439030",
    "session_id": "507f1f77bcf86cd799439031",
    "room_id": "507f1f77bcf86cd799439020",
    "message_id": "external_msg_123",
    "action": "reply_message",
    "data": {
      "message": "Thanks for your interest!",
      "tool_result": "success",
      "execution_time_ms": 245
    },
    "executed_at": "2024-01-01T10:05:00Z",
    "created_at": "2024-01-01T10:05:00Z"
  },
  {
    "id": "507f1f77bcf86cd799439031",
    "session_id": "507f1f77bcf86cd799439031",
    "room_id": "507f1f77bcf86cd799439020",
    "action": "purchase_intent",
    "data": {
      "item_name": "iPhone 12 Pro",
      "intent": "strong_buy_signal",
      "percentage": 85
    },
    "executed_at": "2024-01-01T10:15:00Z",
    "created_at": "2024-01-01T10:15:00Z"
  }
]
```

### Mark Messages as Read

**🔒 Authentication Required**

```http
POST /api/v1/chat/rooms/{id}/read
```

**Response:**
```json
{
  "status": "success",
  "message": "marked as read"
}
```

## Profile Management APIs

### Get Partner Attributes

**🔒 Authentication Required**

```http
GET /api/v1/profile/attributes
```

**Response:**
```json
{
  "chotot_id": "merchant123",
  "chotot_oid": "org456",
  "whatsapp_phone_number_id": "+1234567890"
}
```

### Update Partner Attributes

**🔒 Authentication Required**

```http
PUT /api/v1/profile/attributes
```

**Request Body:**
```json
{
  "chotot_id": "merchant123_updated",
  "chotot_oid": "org456_updated",
  "whatsapp_phone_number_id": "+1234567890",
  "whatsapp_system_token": "encrypted_token_value"
}
```

**Response:**
```json
{
  "status": "success",
  "message": "partner attributes updated successfully"
}
```

## Health Check API

### Health Check

```http
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "service": "chat-bot"
}
```

## Error Handling

### Error Response Format

All API errors follow a consistent format:

```json
{
  "error": "Error description",
  "code": "ERROR_CODE",
  "details": "Additional error details if available"
}
```

### Common Error Codes

| Status Code | Error Type | Description |
|-------------|------------|-------------|
| 400 | `VALIDATION_ERROR` | Invalid request data or missing required fields |
| 401 | `AUTHENTICATION_ERROR` | Missing or invalid authentication token |
| 403 | `AUTHORIZATION_ERROR` | User lacks permission for the resource |
| 404 | `RESOURCE_NOT_FOUND` | Requested resource does not exist |
| 409 | `CONFLICT_ERROR` | Resource conflict (e.g., duplicate email) |
| 422 | `BUSINESS_LOGIC_ERROR` | Request violates business rules |
| 500 | `INTERNAL_ERROR` | Unexpected server error |

### Validation Errors

Field validation errors include specific field information:

```json
{
  "error": "Validation failed",
  "code": "VALIDATION_ERROR",
  "details": {
    "fields": {
      "email": "Email is required and must be valid",
      "name": "Name cannot be empty"
    }
  }
}
```

## Request/Response Examples

### Complete Message Processing Flow

**1. User sends message to Chotot chat**
**2. Chotot forwards to Chat-Bot via HTTP API**

```http
POST /api/v1/messages
x-project-uuid: chotot-project-123
Service: chat-bot
Content-Type: application/json

{
  "room_id": "chotot_room_12345",
  "sender_id": "buyer_user_789",
  "content": "Hi! I'm interested in your iPhone. Can you tell me more about its condition?",
  "created_at": 1640995200000,
  "metadata": {
    "llm": {
      "chat_mode": "product_sales_assistant"
    }
  },
  "partner": {
    "name": "chotot",
    "room_id": "chotot_room_12345",
    "msg_id": "msg_456"
  }
}
```

**3. Chat-Bot processes and responds**

```json
{
  "status": "success",
  "message": "Message processed successfully"
}
```

**Behind the scenes:**
1. Message deduplication check
2. Room information gathering via Chotot API
3. LLM processing with product context
4. AI generates response and may execute tools:
   - `ReplyMessage`: Send response to buyer
   - `PurchaseIntent`: Log buying interest (if detected)
   - `ListProducts`: Show related products (if requested)

### Partner Attribute Management Flow

**1. User logs in**

```http
POST /api/v1/auth/login

{
  "email": "merchant@example.com"
}
```

**2. Get current partner attributes**

```http
GET /api/v1/profile/attributes
Authorization: Bearer <jwt_token>
```

**3. Update Chotot integration settings**

```http
PUT /api/v1/profile/attributes
Authorization: Bearer <jwt_token>

{
  "chotot_id": "updated_merchant_id",
  "chotot_oid": "updated_org_id"
}
```

### Chat Room Management Flow

**1. Get user's active rooms**

```http
GET /api/v1/chat/rooms
Authorization: Bearer <jwt_token>
```

**2. Get specific room messages**

```http
GET /api/v1/chat/rooms/507f1f77bcf86cd799439020/messages?limit=20
Authorization: Bearer <jwt_token>
```

**3. Send manual response**

```http
POST /api/v1/chat/rooms/507f1f77bcf86cd799439020/messages
Authorization: Bearer <jwt_token>

{
  "content": "Thanks for your interest! The iPhone is in excellent condition.",
  "message_type": "text"
}
```

**4. Mark room as read**

```http
POST /api/v1/chat/rooms/507f1f77bcf86cd799439020/read
Authorization: Bearer <jwt_token>
```

The API provides comprehensive functionality for chat-bot operation, user management, and multi-platform partner integration with consistent error handling and authentication patterns.