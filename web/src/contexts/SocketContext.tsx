import React, { createContext, useContext, useEffect, useRef, useState, ReactNode } from 'react';
import { io, Socket } from 'socket.io-client';
import { ChatMessage, TypingIndicator } from '../types';
import { getStoredToken, getStoredUser } from '../utils/auth';

interface SocketContextType {
  socket: Socket | null;
  isConnected: boolean;
  connect: () => void;
  disconnect: () => void;
  joinChannel: (channelId: string) => void;
  leaveChannel: (channelId: string) => void;
  onMessageReceived: (callback: (message: ChatMessage) => void) => void;
  onMessageSent: (callback: (message: ChatMessage) => void) => void;
  onTypingStart: (callback: (typing: TypingIndicator) => void) => void;
  onTypingStop: (callback: (typing: TypingIndicator) => void) => void;
  offMessageReceived: (callback?: (message: ChatMessage) => void) => void;
  offMessageSent: (callback?: (message: ChatMessage) => void) => void;
  offTypingStart: (callback?: (typing: TypingIndicator) => void) => void;
  offTypingStop: (callback?: (typing: TypingIndicator) => void) => void;
}

const SocketContext = createContext<SocketContextType | null>(null);

interface SocketProviderProps {
  children: ReactNode;
}

export const SocketProvider: React.FC<SocketProviderProps> = ({ children }) => {
  const socketRef = useRef<Socket | null>(null);
  const [isConnected, setIsConnected] = useState(false);

  const connectSocket = () => {
    const token = getStoredToken();
    const user = getStoredUser();

    if (!token || !user) {
      // Don't log this as it's normal when user hasn't logged in yet
      return;
    }

    // Disconnect existing socket if any
    if (socketRef.current) {
      socketRef.current.disconnect();
    }

    // Generate or get device/fingerprint IDs
    const deviceId = localStorage.getItem('device_id') || `device_${Date.now()}`;
    const fingerprint = localStorage.getItem('fingerprint') || `fp_${Date.now()}`;

    // Store for future use
    localStorage.setItem('device_id', deviceId);
    localStorage.setItem('fingerprint', fingerprint);

    socketRef.current = io(`${process.env.REACT_APP_WS_URL || 'http://localhost:7070'}/events`, {
      path: '/ws',
      query: {
        token, // Send JWT token as query parameter
        device_id: deviceId,
        fingerprint: fingerprint,
        platform: 'web',
      },
      transports: ['websocket'],
      autoConnect: true,
      reconnection: true,
      reconnectionAttempts: 5,
      reconnectionDelay: 1000,
      timeout: 10000,
    });

    const socket = socketRef.current;

    socket.on('connect', () => {
      console.debug('Socket connected successfully');
      setIsConnected(true);
    });

    socket.on('disconnect', (reason) => {
      console.warn('Socket disconnected:', reason);
      setIsConnected(false);
    });

    socket.on('connect_error', (error) => {
      console.error('Socket connection error:', error);
      setIsConnected(false);
    });

    socket.on('error', (error) => {
      console.error('Socket error:', error);
    });
  };

  const disconnectSocket = () => {
    if (socketRef.current) {
      console.log('Disconnecting socket');
      socketRef.current.disconnect();
      socketRef.current = null;
      setIsConnected(false);
    }
  };

  useEffect(() => {
    // Try to connect on mount if user is already authenticated
    connectSocket();

    return () => {
      disconnectSocket();
    };
  }, []);

  const joinChannel = (channelId: string) => {
    // With the new approach, users are automatically in their own room
    // Messages are sent directly to users based on user_id
    // No explicit join/leave needed
  };

  const leaveChannel = (channelId: string) => {
    // No explicit leave needed with direct user messaging
  };

  const onMessageReceived = (callback: (message: ChatMessage) => void) => {
    if (socketRef.current) {
      socketRef.current.on('message_received', callback);
    }
  };

  const onMessageSent = (callback: (message: ChatMessage) => void) => {
    if (socketRef.current) {
      socketRef.current.on('message_sent', callback);
    }
  };

  const onTypingStart = (callback: (typing: TypingIndicator) => void) => {
    if (socketRef.current) {
      socketRef.current.on('user_typing_start', callback);
    }
  };

  const onTypingStop = (callback: (typing: TypingIndicator) => void) => {
    if (socketRef.current) {
      socketRef.current.on('user_typing_stop', callback);
    }
  };

  const offMessageReceived = (callback?: (message: ChatMessage) => void) => {
    if (socketRef.current) {
      socketRef.current.off('message_received', callback);
    }
  };

  const offMessageSent = (callback?: (message: ChatMessage) => void) => {
    if (socketRef.current) {
      socketRef.current.off('message_sent', callback);
    }
  };

  const offTypingStart = (callback?: (typing: TypingIndicator) => void) => {
    if (socketRef.current) {
      socketRef.current.off('user_typing_start', callback);
    }
  };

  const offTypingStop = (callback?: (typing: TypingIndicator) => void) => {
    if (socketRef.current) {
      socketRef.current.off('user_typing_stop', callback);
    }
  };

  const value: SocketContextType = {
    socket: socketRef.current,
    isConnected,
    connect: connectSocket,
    disconnect: disconnectSocket,
    joinChannel,
    leaveChannel,
    onMessageReceived,
    onMessageSent,
    onTypingStart,
    onTypingStop,
    offMessageReceived,
    offMessageSent,
    offTypingStart,
    offTypingStop,
  };

  return (
    <SocketContext.Provider value={value}>
      {children}
    </SocketContext.Provider>
  );
};

export const useSocket = (): SocketContextType => {
  const context = useContext(SocketContext);
  if (!context) {
    throw new Error('useSocket must be used within a SocketProvider');
  }
  return context;
};
