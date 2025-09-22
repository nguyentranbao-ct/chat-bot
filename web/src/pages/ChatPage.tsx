import React, { useState, useEffect, useCallback } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { Layout } from '../components/Layout';
import ConversationList from '../components/ConversationList';
import ChatWindow from '../components/ChatWindow';
import { useSocket } from '../contexts/SocketContext';
import { api } from '../utils/api';
import { getStoredUser, clearAuth } from '../utils/auth';
import {
  Room,
  ChatMessage,
  SendMessageRequest,
  User,
  TypingIndicator,
} from '../types';

const ChatPage: React.FC = () => {
  const navigate = useNavigate();
  const { roomId } = useParams<{ roomId?: string }>();
  const socket = useSocket();

  const [user, setUser] = useState<User | null>(null);
  const [rooms, setRooms] = useState<Room[]>([]);
  const [selectedRoom, setSelectedRoom] = useState<Room | null>(null);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [isTyping, setIsTyping] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');

  // Define callback functions first
  const loadRooms = useCallback(async (forceRefresh = false) => {
    try {
      const roomData = await api.getRooms();
      setRooms(roomData || []);
    } catch (err: any) {
      console.error('Failed to load rooms:', err);
      setError('Failed to load conversations');
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Check if a room exists locally before refreshing
  const findRoomById = useCallback((roomId: string) => {
    return rooms.find(room => room.id === roomId);
  }, [rooms]);

  const loadMessages = useCallback(async (roomId: string) => {
    try {
      const messageData = await api.getRoomMessages(roomId);
      // Sort messages by created_at to ensure proper ordering (oldest first)
      const sortedMessages = (messageData || []).sort((a, b) =>
        new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
      );
      setMessages(sortedMessages);
    } catch (err: any) {
      console.error('Failed to load messages:', err);
      setError('Failed to load messages');
      setMessages([]); // Set empty array on error
    }
  }, []);

  const handleSendMessage = useCallback(async (request: SendMessageRequest) => {
    if (!selectedRoom || !user) return;

    try {
      await api.sendMessage(selectedRoom.id, request);
      // Don't add message to local state - let socket handle it to prevent duplicates
      // Don't call loadRooms here - let socket handle updates
    } catch (err: any) {
      console.error('Failed to send message:', err);
      setError('Failed to send message');
    }
  }, [selectedRoom, user]);

  const handleMarkAsRead = useCallback(async (messageId: string) => {
    if (!selectedRoom) return;

    try {
      await api.markAsRead(selectedRoom.id, messageId);
    } catch (err: any) {
      console.error('Failed to mark as read:', err);
    }
  }, [selectedRoom]);

  // Initialize user and data
  useEffect(() => {
    const storedUser = getStoredUser();
    if (!storedUser) {
      navigate('/login');
      return;
    }

    setUser(storedUser);
    loadRooms();
  }, [navigate, loadRooms]);

  // Handle URL room parameter
  useEffect(() => {
    if (roomId && rooms.length > 0) {
      const room = rooms.find(c => c.id === roomId);
      if (room && (!selectedRoom || selectedRoom.id !== roomId)) {
        setSelectedRoom(room);
      }
    }
  }, [roomId, rooms, selectedRoom]);

  // Set up socket listeners - stable references to avoid re-renders
  useEffect(() => {
    const handleMessageReceived = (message: ChatMessage) => {
      setMessages((prev) => {
        // Only add if it's for current room and not already exists
        if (selectedRoom?.id === message.room_id && !prev.find(m => m.id === message.id)) {
          const newMessages = [...prev, message];
          // Sort messages to maintain chronological order
          return newMessages.sort((a, b) =>
            new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
          );
        }
        return prev;
      });

      // Optimized refresh logic: only refresh if this is a new channel
      const existingRoom = findRoomById(message.room_id);
      if (!existingRoom) {
        // This is a new channel, refresh the entire list
        console.log('New channel detected, refreshing room list');
        loadRooms(true);
      } else {
        // Update last message info for existing room without full refresh
        setRooms(prev => prev.map(room =>
          room.id === message.room_id
            ? {
                ...room,
                last_message_at: message.created_at,
                unread_count: selectedRoom?.id === message.room_id ? 0 : room.unread_count + 1
              }
            : room
        ));
      }
    };

    const handleMessageSent = (message: ChatMessage) => {
      setMessages((prev) => {
        if (selectedRoom?.id === message.room_id && !prev.find(m => m.id === message.id)) {
          const newMessages = [...prev, message];
          // Sort messages to maintain chronological order
          return newMessages.sort((a, b) =>
            new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
          );
        }
        return prev;
      });

      // Update room data for sent messages (typically from current user)
      setRooms(prev => prev.map(room =>
        room.id === message.room_id
          ? { ...room, last_message_at: message.created_at }
          : room
      ));
    };

    const handleTypingStart = (typing: TypingIndicator) => {
      if (selectedRoom?.id === typing.room_id && typing.user_id !== user?.id) {
        setIsTyping(true);
      }
    };

    const handleTypingStop = (typing: TypingIndicator) => {
      if (selectedRoom?.id === typing.room_id && typing.user_id !== user?.id) {
        setIsTyping(false);
      }
    };

    // Only set up listeners if socket is available
    if (socket.isConnected) {
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
    }
  }, [selectedRoom?.id, user?.id, socket.isConnected, findRoomById, loadRooms]);

  // Join/leave rooms when selection changes
  useEffect(() => {
    if (selectedRoom) {
      // Only load messages, don't depend on socket connection for this
      loadMessages(selectedRoom.id);

      // Try to join socket room if connected
      if (socket.isConnected) {
        socket.joinRoom(selectedRoom.id);
      }

      return () => {
        if (socket.isConnected) {
          socket.leaveRoom(selectedRoom.id);
        }
      };
    }
  }, [selectedRoom]);

  const handleRoomSelect = useCallback((room: Room) => {
    setSelectedRoom(room);
    setMessages([]);
    setIsTyping(false);
    // Update URL to include room ID
    navigate(`/chat/${room.id}`);
  }, [navigate]);

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
          rooms={rooms}
          selectedRoomId={selectedRoom?.id}
          onRoomSelect={handleRoomSelect}
        />

        {selectedRoom ? (
          <ChatWindow
            room={selectedRoom}
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
