import React, { useState, useEffect, useCallback, useRef } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { Layout } from '../components/Layout';
import { Header } from '../components/Header';
import { PartnerAttributesModal } from '../components/PartnerAttributesModal';
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
  const [showPartnerModal, setShowPartnerModal] = useState(false);
  const [isInitialSetup, setIsInitialSetup] = useState(false);

  // Define callback functions first
  // Prevent redundant reloads when navigating between room routes that remount the component
  const roomsLoadedRef = useRef(false);
  const loadRooms = useCallback(async (forceRefresh = false) => {
    if (roomsLoadedRef.current && !forceRefresh) {
      // Already loaded once and no forced refresh requested
      setIsLoading(false);
      return;
    }
    try {
      const roomData = await api.getRooms();
      setRooms(roomData || []);
      roomsLoadedRef.current = true;
    } catch (err: any) {
      console.error('Failed to load rooms:', err);
      setError('Failed to load conversations');
    } finally {
      setIsLoading(false);
    }
  }, []);


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

  const handleMarkAsRead = useCallback(async (message: ChatMessage) => {
    if (!selectedRoom) return;

    try {
      if (selectedRoom.last_read_at && new Date(message.created_at) <= new Date(selectedRoom.last_read_at)) {
        // Already marked as read
        return;
      }
      await api.markAsRead(selectedRoom.id);
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
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [navigate, loadRooms]);

  // Check partner attributes only once when component mounts and user is set
  useEffect(() => {
    const checkPartnerAttributes = async () => {
      try {
        const attributes = await api.getPartnerAttributes();
        const hasAnyAttribute = Object.values(attributes).some(value => value && value.trim() !== '');

        if (!hasAnyAttribute) {
          setIsInitialSetup(true);
          setShowPartnerModal(true);
        }
      } catch (err) {
        // If we can't fetch attributes or they don't exist, show the modal
        console.log('No partner attributes found, showing setup modal');
        setIsInitialSetup(true);
        setShowPartnerModal(true);
      }
    };

    checkPartnerAttributes();
  }, []);

  // Handle URL room parameter
  useEffect(() => {
    if (roomId && rooms.length > 0) {
      const room = rooms.find(c => c.id === roomId);
      if (room && (!selectedRoom || selectedRoom.id !== roomId)) {
        setSelectedRoom(room);
      }
    }
  }, [roomId, rooms, selectedRoom,]);

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

      // Update room info without calling API - we already have the data
      setRooms(prev => {
        const existingRoomIndex = prev.findIndex(room => room.id === message.room_id);
        if (existingRoomIndex === -1) {
          // This is genuinely a new room, refresh the room list to get it
          console.log('New room detected, refreshing room list...');
          loadRooms(true);
          return prev;
        }

        // Update existing room
        return prev.map(room =>
          room.id === message.room_id
            ? {
              ...room,
              last_message_at: message.created_at,
              last_message_content: message.content,
              unread_count: selectedRoom?.id === message.room_id ||
                message.sender_id === user?.id ?
                0 : room.unread_count + 1
            }
            : room
        );
      });
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
  }, [selectedRoom?.id, user?.id, socket, loadRooms]);

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
  }, [selectedRoom, socket, loadMessages]);

  const handleRoomSelect = useCallback((room: Room) => {
    if (selectedRoom?.id === room.id) {
      return;
    }

    setSelectedRoom(room);
    setMessages([]);
    setIsTyping(false);

    // Immediately clear unread count in local state
    if (room.unread_count > 0) {
      setRooms(prev => prev.map(r =>
        r.id === room.id ? { ...r, unread_count: 0 } : r
      ));
    }

    // Update URL to include room ID
    navigate(`/chat/${room.id}`);
  }, [navigate, selectedRoom]);

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

  const handlePartnerSettings = () => {
    setIsInitialSetup(false);
    setShowPartnerModal(true);
  };

  const handlePartnerModalClose = () => {
    setShowPartnerModal(false);
  };

  if (isLoading) {
    return (
      <Layout
        header={user && (
          <Header
            user={user}
            onPartnerSettings={handlePartnerSettings}
            onLogout={handleLogout}
          />
        )}
      >
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
    <Layout
      header={user && (
        <Header
          user={user}
          onPartnerSettings={handlePartnerSettings}
          onLogout={handleLogout}
        />
      )}
    >
      <div className="h-full flex relative">
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

      {/* Partner Attributes Modal */}
      {showPartnerModal &&
        <PartnerAttributesModal
          isOpen={showPartnerModal}
          onClose={handlePartnerModalClose}
          showSkipOption={isInitialSetup}
          title={isInitialSetup ? "Welcome! Set up your partner integrations" : "Partner Integration Settings"}
        />
      }
    </Layout>
  );
};

export default ChatPage;
