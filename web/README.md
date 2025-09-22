# Moshee Chat Web Frontend

React-based frontend for the chat-bot application, designed to match the Moshee interface style.

## Features

- **Email-only Login**: Simple authentication using email addresses
- **Real-time Chat**: Live messaging with Socket.IO integration
- **AI Assistant Panel**: AI-powered suggestions and responses
- **Responsive Design**: Optimized for desktop and mobile
- **Block-style Messages**: Support for rich message formatting (Slack-like)
- **Typing Indicators**: Real-time typing status
- **Unread Counters**: Track unread messages per conversation

## Technologies

- React 18 with TypeScript
- Tailwind CSS for styling
- Socket.IO for real-time communication
- React Router for navigation
- Axios for API communication

## Setup

1. Install dependencies:

   ```bash
   npm install
   ```

2. Copy environment configuration:

   ```bash
   cp .env.example .env
   ```

3. Update `.env` with your API URLs:

   ```
   REACT_APP_API_URL=http://localhost:8080/api/v1
   REACT_APP_WS_URL=ws://localhost:8080
   ```

4. Start development server:
   ```bash
   npm start
   ```

The application will be available at http://localhost:3000

## Build for Production

```bash
npm run build
```

## Component Structure

- `LoginPage`: Email-based authentication
- `ChatPage`: Main chat interface
- `ConversationList`: Room/conversation sidebar
- `ChatWindow`: Main chat area with message display
- `MessageInput`: Message composition input
- `AIAssistant`: AI suggestions panel
- `Layout`: Base layout component

## API Integration

The frontend integrates with the Go backend API:

- `POST /api/v1/auth/login` - Email login
- `GET /api/v1/chat/rooms` - Get user rooms
- `GET /api/v1/chat/rooms/:id/messages` - Get room messages
- `POST /api/v1/chat/rooms/:id/messages` - Send message
- `POST /api/v1/chat/rooms/:id/read` - Mark as read

## Socket.IO Events

Real-time events handled:

- `message_received` - New message from others
- `message_sent` - Confirmation of sent message
- `user_typing_start` - User started typing
- `user_typing_stop` - User stopped typing

## Design System

Following the Moshee design language:

- Clean, minimal interface
- Blue primary colors (#2563eb, #3b82f6)
- Rounded corners and soft shadows
- Clear typography hierarchy
- Intuitive icons and interactions
