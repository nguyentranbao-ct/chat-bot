import React, { useState, useRef, useEffect } from 'react';
import { Channel, ChatMessage, SendMessageRequest } from '../types';
import MessageInput from './MessageInput';
import AIAssistant from './AIAssistant';

interface ChatWindowProps {
  channel: Channel;
  messages: ChatMessage[];
  currentUserId: string;
  onSendMessage: (message: SendMessageRequest) => void;
  onMarkAsRead: (messageId: string) => void;
  isTyping?: boolean;
}

const ChatWindow: React.FC<ChatWindowProps> = ({
  channel,
  messages,
  currentUserId,
  onSendMessage,
  onMarkAsRead,
  isTyping = false,
}) => {
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const messagesContainerRef = useRef<HTMLDivElement>(null);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  // Mark last message as read when component mounts or messages change
  useEffect(() => {
    if (messages && messages.length > 0) {
      const lastMessage = messages[messages.length - 1];
      if (lastMessage.sender_id !== currentUserId) {
        onMarkAsRead(lastMessage.id);
      }
    }
  }, [messages, currentUserId, onMarkAsRead]);

  const formatMessageTime = (timestamp: string) => {
    const date = new Date(timestamp);
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  };

  const isMessageFromCurrentUser = (message: ChatMessage) => {
    return message.sender_id === currentUserId;
  };

  const renderMessageContent = (message: ChatMessage) => {
    // If message has blocks, render them
    if (message.blocks && Array.isArray(message.blocks) && message.blocks.length > 0) {
      return (
        <div className="space-y-2">
          {message.blocks.map((block, index) => (
            <div key={index} className="block">
              {block.type === 'text' && <span>{block.content}</span>}
              {block.type === 'code' && (
                <pre className="bg-gray-100 p-2 rounded text-sm overflow-x-auto">
                  <code>{block.content}</code>
                </pre>
              )}
              {block.type === 'link' && (
                <a
                  href={block.content}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-500 hover:underline"
                >
                  {block.content}
                </a>
              )}
            </div>
          ))}
        </div>
      );
    }

    // Otherwise render plain content
    return <span>{message.content || ''}</span>;
  };

  return (
    <div className="flex-1 flex">
      {/* Main Chat Area */}
      <div className="flex-1 flex flex-col">
        {/* Header */}
        <div className="bg-white border-b border-gray-200 p-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-3">
              <div className="w-8 h-8 bg-blue-100 rounded-full flex items-center justify-center">
                <span className="text-blue-600 font-medium text-sm">
                  {channel.name?.substring(0, 2).toUpperCase() || 'CH'}
                </span>
              </div>
              <div>
                <h2 className="text-lg font-semibold text-gray-900">
                  {channel.name || 'Unnamed Channel'}
                </h2>
                <div className="text-sm text-gray-500">
                  {channel.item_name && <span>{channel.item_name}</span>}
                  {channel.item_price && <span> • {channel.item_price}</span>}
                </div>
              </div>
            </div>

            <div className="flex items-center space-x-2">
              <button className="p-2 text-gray-400 hover:text-gray-600">
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 5a2 2 0 012-2h3.28a1 1 0 01.948.684l1.498 4.493a1 1 0 01-.502 1.21L6.3 11.043c-.34.17-.34.65-.005.892C7.784 13.639 10.361 16.216 12.065 17.705c.242.335.722.335.892-.005l1.656-3.924a1 1 0 011.21-.502l4.493 1.498a1 1 0 01.684.949V19a2 2 0 01-2 2h-1C9.716 21 3 14.284 3 6V5z" />
                </svg>
              </button>
              <button className="p-2 text-gray-400 hover:text-gray-600">
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z" />
                </svg>
              </button>
            </div>
          </div>
        </div>

        {/* Messages */}
        <div
          ref={messagesContainerRef}
          className="flex-1 overflow-y-auto p-4 space-y-4 bg-gray-50"
        >
          {!messages || messages.length === 0 ? (
            <div className="text-center text-gray-500 mt-8">
              No messages yet. Start the conversation!
            </div>
          ) : (
            messages.map((message, index) => {
              const isFromCurrentUser = isMessageFromCurrentUser(message);
              const showTimestamp = index === 0 ||
                new Date(message.created_at).getTime() - new Date(messages[index - 1].created_at).getTime() > 300000; // 5 minutes

              return (
                <div key={message.id} className="space-y-2">
                  {showTimestamp && (
                    <div className="text-center text-xs text-gray-500">
                      {formatMessageTime(message.created_at)}
                    </div>
                  )}

                  <div className={`flex ${isFromCurrentUser ? 'justify-end' : 'justify-start'}`}>
                    <div
                      className={`message-bubble ${isFromCurrentUser ? 'sent' : 'received'
                        }`}
                    >
                      {renderMessageContent(message)}

                      {message.is_edited && (
                        <div className="text-xs opacity-75 mt-1">
                          (edited)
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              );
            })
          )}

          {isTyping && (
            <div className="flex justify-start">
              <div className="message-bubble received">
                <div className="flex space-x-1">
                  <div className="w-2 h-2 bg-gray-400 rounded-full animate-pulse"></div>
                  <div className="w-2 h-2 bg-gray-400 rounded-full animate-pulse" style={{ animationDelay: '0.1s' }}></div>
                  <div className="w-2 h-2 bg-gray-400 rounded-full animate-pulse" style={{ animationDelay: '0.2s' }}></div>
                </div>
              </div>
            </div>
          )}

          <div ref={messagesEndRef} />
        </div>

        {/* Message Input */}
        <MessageInput onSendMessage={onSendMessage} />
      </div>

      {/* AI Assistant Panel */}
      <AIAssistant channel={channel} />
    </div>
  );
};

export default ChatWindow;
