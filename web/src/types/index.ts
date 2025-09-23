export interface User {
  id: string;
  name?: string;
  email: string;
  chotot_id?: string;
  chotot_oid?: string;
  chat_mode?: string;
  created_at: string;
  updated_at: string;
  last_login_at?: string;
  profile_setup_at?: string;
  is_active: boolean;
}

export interface LoginRequest {
  email: string;
}

export interface LoginResponse {
  token: string;
  user: User;
  expires_at: string;
}

export interface ProfileUpdateRequest {
  name?: string;
  chotot_id?: string;
  chotot_oid?: string;
  chat_mode?: string;
}

export interface Room {
  id: string;
  external_room_id: string;
  name: string;
  item_name?: string;
  item_price?: string;
  context?: string;
  type: string;
  created_at: string;
  updated_at: string;
  last_read_at?: string;
  last_message_at?: string;
  last_message_content: string;
  is_archived: boolean;
  unread_count: number;
}

export interface RoomMemberInfo {
  id: string;
  user_id: string;
  role: string;
  joined_at: string;
}

export interface MessageBlock {
  type: string;
  content: string;
  style?: Record<string, any>;
}

export interface ChatMessage {
  id: string;
  room_id: string;
  external_room_id: string;
  sender_id: string;
  message_type: string;
  content: string;
  blocks?: MessageBlock[];
  created_at: string;
  updated_at: string;
  is_edited: boolean;
  edited_at?: string;
  is_deleted: boolean;
  delivery_status: string;
  metadata: {
    source: string;
    is_from_bot: boolean;
    original_timestamp?: number;
    custom_data?: Record<string, any>;
  };
}

export interface SendMessageRequest {
  content: string;
  message_type?: string;
  blocks?: MessageBlock[];
  metadata?: Record<string, any>;
}

export interface MessageEvent {
  id: string;
  room_id: string;
  event_type: string;
  message_id?: string;
  user_id: string;
  event_data: Record<string, any>;
  created_at: string;
  expires_at: string;
}

export interface TypingIndicator {
  room_id: string;
  user_id: string;
  is_typing: boolean;
}

export interface SocketEvents {
  message_sent: ChatMessage;
  message_received: ChatMessage;
  user_typing_start: TypingIndicator;
  user_typing_stop: TypingIndicator;
  room_updated: Room;
}
