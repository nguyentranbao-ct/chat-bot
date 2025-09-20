# Chat API Documentation

## Get Messages in a Channel

```
GET /api/v1/internal/user_channel/{user_id}/{channel_id}/messages
```

**Query Parameters:**

- `limit` (int, optional): Max messages (default: 100)
- `before_ts` (int64, optional): Unix ms timestamp

**Headers:**

- `x-project-uuid`: Project UUID
- `Service`: Service identifier

**Example:**

```bash
curl --location 'http://chat-api.chat/api/v1/internal/user_channel/$USER_ID/$CHANNEL_ID/messages?limit=100&before_ts=$UNIX_MS' \
--header 'x-project-uuid: 16f38160-3afa-4707-b8cb-354d2cbf1590' \
--header 'Service: chat-bot'
```

**Response (200):**

```json
{
  "code": 200,
  "data": [
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
      "type": "string",
      "metadata": {},
      "is_edited": false
    }
  ]
}
```

## Get User Channels in a Channel

```
GET /api/v1/internal/channels/{channel_id}/user_channels
```

**Headers:**

- `x-project-uuid`: Project UUID
- `Service`: Service identifier

**Example:**

```bash
curl --location 'http://chat-api.chat/api/v1/internal/channels/30APeUsHYrde6THBN0ST7QgKcyA/user_channels' \
--header 'x-project-uuid: 16f38160-3afa-4707-b8cb-354d2cbf1590' \
--header 'Service: chat-bot'
```

**Response (200):**

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

## Send Message to a Channel

POST /api/v1/internal/messages

```bash
curl --location 'https://chat-api.chat/api/v1/internal/messages' \
--header 'x-project-uuid: 16f38160-3afa-4707-b8cb-354d2cbf1590' \
--header 'Service: chat-bot' \
--header 'Content-Type: application/json' \
--data '{
    "channel_id": "CHANNEL_ID",
    "sender_id": "SENDER_ID",
    "message": "MESSAGE",
    "type": "text"
}'
```
