import axios, { AxiosInstance, AxiosResponse } from 'axios';
import {
  LoginRequest,
  LoginResponse,
  User,
  ProfileUpdateRequest,
  Room,
  RoomMemberInfo,
  ChatMessage,
  SendMessageRequest,
  MessageEvent,
} from '../types';

class ApiClient {
  private client: AxiosInstance;

  constructor() {
    this.client = axios.create({
      baseURL: process.env.REACT_APP_API_URL || 'http://localhost:8080/api/v1',
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // Add auth token to requests
    this.client.interceptors.request.use((config) => {
      const token = localStorage.getItem('auth_token');
      if (token) {
        config.headers.Authorization = `Bearer ${token}`;
      }
      return config;
    });

    // Handle auth errors
    this.client.interceptors.response.use(
      (response) => response,
      (error) => {
        if (error.response?.status === 401) {
          localStorage.removeItem('auth_token');
          localStorage.removeItem('user');
          window.location.href = '/login';
        }
        return Promise.reject(error);
      },
    );
  }

  // Auth endpoints
  async login(request: LoginRequest): Promise<LoginResponse> {
    const response: AxiosResponse<LoginResponse> = await this.client.post(
      '/auth/login',
      request,
    );
    return response.data;
  }

  async logout(): Promise<void> {
    await this.client.post('/auth/logout');
  }

  async getProfile(): Promise<User> {
    const response: AxiosResponse<User> = await this.client.get('/auth/me');
    return response.data;
  }

  async updateProfile(request: ProfileUpdateRequest): Promise<User> {
    const response: AxiosResponse<User> = await this.client.put(
      '/auth/profile',
      request,
    );
    return response.data;
  }

  // Chat endpoints
  async getRooms(): Promise<Room[]> {
    const response: AxiosResponse<Room[]> = await this.client.get(
      '/chat/rooms',
    );
    return response.data;
  }

  async getRoomMembers(roomId: string): Promise<RoomMemberInfo[]> {
    const response: AxiosResponse<RoomMemberInfo[]> = await this.client.get(
      `/chat/rooms/${roomId}/members`,
    );
    return response.data;
  }

  async getRoomMessages(
    roomId: string,
    limit = 50,
    before?: string,
  ): Promise<ChatMessage[]> {
    const params = new URLSearchParams({
      limit: limit.toString(),
      ...(before && { before }),
    });
    const response: AxiosResponse<ChatMessage[]> = await this.client.get(
      `/chat/rooms/${roomId}/messages?${params}`,
    );
    return response.data;
  }

  async sendMessage(
    roomId: string,
    request: SendMessageRequest,
  ): Promise<ChatMessage> {
    const response: AxiosResponse<ChatMessage> = await this.client.post(
      `/chat/rooms/${roomId}/messages`,
      request,
    );
    return response.data;
  }

  async getRoomEvents(
    roomId: string,
    sinceTime: string,
  ): Promise<MessageEvent[]> {
    const response: AxiosResponse<MessageEvent[]> = await this.client.get(
      `/chat/rooms/${roomId}/events?since=${sinceTime}`,
    );
    return response.data;
  }

  async markAsRead(roomId: string, lastReadMessageId: string): Promise<void> {
    await this.client.post(`/chat/rooms/${roomId}/read`, {
      message_id: lastReadMessageId,
    });
  }

  async setTyping(roomId: string, isTyping: boolean): Promise<void> {
    await this.client.post(`/chat/rooms/${roomId}/typing`, {
      is_typing: isTyping,
    });
  }
}

export const api = new ApiClient();
