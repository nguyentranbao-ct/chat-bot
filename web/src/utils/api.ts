import axios, { AxiosInstance, AxiosResponse } from 'axios';
import {
  LoginRequest,
  LoginResponse,
  User,
  ProfileUpdateRequest,
  Channel,
  ChannelMember,
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
      }
    );
  }

  // Auth endpoints
  async login(request: LoginRequest): Promise<LoginResponse> {
    const response: AxiosResponse<LoginResponse> = await this.client.post('/auth/login', request);
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
    const response: AxiosResponse<User> = await this.client.put('/auth/profile', request);
    return response.data;
  }

  // Chat endpoints
  async getChannels(): Promise<Channel[]> {
    const response: AxiosResponse<Channel[]> = await this.client.get('/chat/channels');
    return response.data;
  }

  async getChannelMembers(channelId: string): Promise<ChannelMember[]> {
    const response: AxiosResponse<ChannelMember[]> = await this.client.get(`/chat/channels/${channelId}/members`);
    return response.data;
  }

  async getChannelMessages(channelId: string, limit = 50, before?: string): Promise<ChatMessage[]> {
    const params = new URLSearchParams({
      limit: limit.toString(),
      ...(before && { before }),
    });
    const response: AxiosResponse<ChatMessage[]> = await this.client.get(
      `/chat/channels/${channelId}/messages?${params}`
    );
    return response.data;
  }

  async sendMessage(channelId: string, request: SendMessageRequest): Promise<ChatMessage> {
    const response: AxiosResponse<ChatMessage> = await this.client.post(
      `/chat/channels/${channelId}/messages`,
      request
    );
    return response.data;
  }

  async getChannelEvents(channelId: string, sinceTime: string): Promise<MessageEvent[]> {
    const response: AxiosResponse<MessageEvent[]> = await this.client.get(
      `/chat/channels/${channelId}/events?since=${sinceTime}`
    );
    return response.data;
  }

  async markAsRead(channelId: string, lastReadMessageId: string): Promise<void> {
    await this.client.post(`/chat/channels/${channelId}/read`, {
      message_id: lastReadMessageId,
    });
  }

  async setTyping(channelId: string, isTyping: boolean): Promise<void> {
    await this.client.post(`/chat/channels/${channelId}/typing`, {
      is_typing: isTyping,
    });
  }
}

export const api = new ApiClient();