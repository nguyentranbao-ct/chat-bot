import { useEffect, useRef, useState } from 'react';
import { io, Socket } from 'socket.io-client';
import { SocketEvents, ChatMessage, TypingIndicator } from '../types';
import { getStoredToken } from '../utils/auth';

export const useSocket = () => {
  const socketRef = useRef<Socket | null>(null);
  const [isConnected, setIsConnected] = useState(false);

  useEffect(() => {
    const token = getStoredToken();
    if (!token) return;

    // Initialize socket connection
    socketRef.current = io(process.env.REACT_APP_WS_URL || 'ws://localhost:8080', {
      auth: {
        token,
      },
      transports: ['websocket'],
    });

    const socket = socketRef.current;

    socket.on('connect', () => {
      console.log('Socket connected');
      setIsConnected(true);
    });

    socket.on('disconnect', () => {
      console.log('Socket disconnected');
      setIsConnected(false);
    });

    socket.on('connect_error', (error) => {
      console.error('Socket connection error:', error);
      setIsConnected(false);
    });

    return () => {
      socket.disconnect();
    };
  }, []);

  const joinChannel = (channelId: string) => {
    if (socketRef.current) {
      socketRef.current.emit('join_channel', channelId);
    }
  };

  const leaveChannel = (channelId: string) => {
    if (socketRef.current) {
      socketRef.current.emit('leave_channel', channelId);
    }
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

  return {
    isConnected,
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
};