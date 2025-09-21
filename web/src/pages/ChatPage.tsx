import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Layout } from '../components/Layout';
import ConversationList from '../components/ConversationList';
import ChatWindow from '../components/ChatWindow';
import { useSocket } from '../hooks/useSocket';
import { api } from '../utils/api';
import { getStoredUser, clearAuth } from '../utils/auth';
import {
  Channel,
  ChatMessage,
  SendMessageRequest,
  User,
  TypingIndicator,
} from '../types';

const ChatPage: React.FC = () => {
  const navigate = useNavigate();
  const socket = useSocket();

  const [user, setUser] = useState<User | null>(null);
  const [channels, setChannels] = useState<Channel[]>([]);
  const [selectedChannel, setSelectedChannel] = useState<Channel | null>(null);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [isTyping, setIsTyping] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');

  // Initialize user and data
  useEffect(() => {
    const storedUser = getStoredUser();
    if (!storedUser) {
      navigate('/login');
      return;
    }

    setUser(storedUser);
    loadChannels();
  }, [navigate]);

  // Set up socket listeners
  useEffect(() => {
    const handleMessageReceived = (message: ChatMessage) => {
      if (selectedChannel && message.channel_id === selectedChannel.id) {
        setMessages((prev) => [...prev, message]);
      }
      // Update channel list to show new message
      loadChannels();
    };

    const handleMessageSent = (message: ChatMessage) => {
      if (selectedChannel && message.channel_id === selectedChannel.id) {
        setMessages((prev) => [...prev, message]);
      }
    };

    const handleTypingStart = (typing: TypingIndicator) => {
      if (selectedChannel && typing.channel_id === selectedChannel.id && typing.user_id !== user?.id) {
        setIsTyping(true);
      }
    };

    const handleTypingStop = (typing: TypingIndicator) => {
      if (selectedChannel && typing.channel_id === selectedChannel.id && typing.user_id !== user?.id) {
        setIsTyping(false);
      }
    };

    socket.onMessageReceived(handleMessageReceived);
    socket.onMessageSent(handleMessageSent);
    socket.onTypingStart(handleTypingStart);
    socket.onTypingStop(handleTypingStop);

    return () => {
      socket.offMessageReceived(handleMessageReceived);
      socket.offMessageSent(handleMessageSent);
      socket.offTypingStart(handleTypingStart);
      socket.offTypingStop(handleTypingStop);
    };
  }, [socket, selectedChannel, user]);

  // Join/leave channels when selection changes
  useEffect(() => {
    if (selectedChannel) {
      socket.joinChannel(selectedChannel.id);
      loadMessages(selectedChannel.id);

      return () => {
        socket.leaveChannel(selectedChannel.id);
      };
    }
  }, [selectedChannel, socket]);

  const loadChannels = async () => {
    try {
      const channelData = await api.getChannels();
      setChannels(channelData);
    } catch (err: any) {
      console.error('Failed to load channels:', err);
      setError('Failed to load conversations');
    } finally {
      setIsLoading(false);
    }
  };

  const loadMessages = async (channelId: string) => {
    try {
      const messageData = await api.getChannelMessages(channelId);
      setMessages(messageData);
    } catch (err: any) {
      console.error('Failed to load messages:', err);
      setError('Failed to load messages');
    }
  };

  const handleChannelSelect = (channel: Channel) => {
    setSelectedChannel(channel);
    setMessages([]);
    setIsTyping(false);
  };

  const handleSendMessage = async (request: SendMessageRequest) => {
    if (!selectedChannel || !user) return;

    try {
      const message = await api.sendMessage(selectedChannel.id, request);
      setMessages((prev) => [...prev, message]);
      loadChannels(); // Refresh to update last message time
    } catch (err: any) {
      console.error('Failed to send message:', err);
      setError('Failed to send message');
    }
  };

  const handleMarkAsRead = async (messageId: string) => {
    if (!selectedChannel) return;

    try {
      await api.markAsRead(selectedChannel.id, messageId);
    } catch (err: any) {
      console.error('Failed to mark as read:', err);
    }
  };

  const handleLogout = async () => {
    try {
      await api.logout();
    } catch (err: any) {
      console.error('Logout error:', err);
    } finally {
      clearAuth();
      navigate('/login');
    }
  };

  if (isLoading) {
    return (
      <Layout>
        <div className="h-full flex items-center justify-center">
          <div className="flex items-center space-x-2">
            <svg className="animate-spin h-6 w-6 text-blue-600" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            <span className="text-gray-600">Loading conversations...</span>
          </div>
        </div>
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="h-full flex">
        <ConversationList
          channels={channels}
          selectedChannelId={selectedChannel?.id}
          onChannelSelect={handleChannelSelect}
        />

        {selectedChannel ? (
          <ChatWindow
            channel={selectedChannel}
            messages={messages}
            currentUserId={user?.id || ''}
            onSendMessage={handleSendMessage}
            onMarkAsRead={handleMarkAsRead}
            isTyping={isTyping}
          />
        ) : (
          <div className="flex-1 flex items-center justify-center bg-gray-50">
            <div className="text-center">
              <div className="w-16 h-16 mx-auto bg-gray-200 rounded-full flex items-center justify-center mb-4">
                <svg className="w-8 h-8 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
                </svg>
              </div>
              <h3 className="text-lg font-medium text-gray-900 mb-2">
                Welcome to Moshee
              </h3>
              <p className="text-gray-500">
                Select a conversation to start messaging
              </p>
            </div>
          </div>
        )}

        {/* User Menu (floating) */}
        <div className="absolute top-4 right-4">
          <div className="relative">
            <button
              onClick={handleLogout}
              className="p-2 bg-white border border-gray-200 rounded-lg text-gray-600 hover:text-gray-900 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500"
              title="Logout"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
              </svg>
            </button>
          </div>
        </div>

        {/* Error toast */}
        {error && (
          <div className="absolute bottom-4 right-4 bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded-lg shadow-lg">
            <div className="flex items-center">
              <svg className="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <span>{error}</span>
              <button
                onClick={() => setError('')}
                className="ml-4 text-red-700 hover:text-red-900"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
          </div>
        )}
      </div>
    </Layout>
  );
};

export default ChatPage;