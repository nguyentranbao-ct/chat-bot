import React from 'react';
import { Room } from '../types';

interface ConversationListProps {
  rooms: Room[];
  selectedRoomId?: string;
  onRoomSelect: (room: Room) => void;
}

const ConversationList: React.FC<ConversationListProps> = ({
  rooms,
  selectedRoomId,
  onRoomSelect,
}) => {
  const formatLastMessageTime = (timestamp?: string) => {
    if (!timestamp) return '';

    const date = new Date(timestamp);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 1) return 'now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    if (diffDays < 7) return `${diffDays}d ago`;

    return date.toLocaleDateString();
  };


  return (
    <div className="w-80 bg-white border-r border-gray-200 flex flex-col">
      {/* Header */}
      <div className="p-4 border-b border-gray-200">
        <div className="flex items-center justify-between">
          <h1 className="text-xl font-semibold text-gray-900">Messages</h1>
          <div className="flex items-center space-x-2">
            <button className="p-1 text-gray-400 hover:text-gray-600">
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z" />
              </svg>
            </button>
          </div>
        </div>
      </div>

      {/* Filters */}
      <div className="p-3 border-b border-gray-100">
        <div className="flex space-x-4 text-sm">
          <button className="flex items-center space-x-1 text-gray-900 font-medium">
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <span>Needs Input</span>
          </button>
          <button className="flex items-center space-x-1 text-gray-500">
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
            </svg>
            <span>Unread Only</span>
          </button>
        </div>
      </div>

      {/* Conversation List */}
      <div className="flex-1 overflow-y-auto">
        {rooms.length === 0 ? (
          <div className="p-4 text-center text-gray-500">
            No conversations yet
          </div>
        ) : (
          rooms.map((room) => (
            <div
              key={room.id}
              className={`conversation-list-item ${selectedRoomId === room.id ? 'active' : ''
                }`}
              onClick={() => onRoomSelect(room)}
            >
              <div className="flex items-start space-x-3">
                {/* Avatar */}
                <div className="w-10 h-10 bg-blue-100 rounded-full flex items-center justify-center flex-shrink-0">
                  <span className="text-blue-600 font-medium text-sm">
                    {room.name?.substring(0, 2).toUpperCase() || 'CH'}
                  </span>
                </div>

                {/* Content */}
                <div className="flex-1 min-w-0">
                  <div className="flex items-center justify-between">
                    <h3 className="text-sm font-medium text-gray-900 truncate">
                      {room.name || 'Unnamed Room'}
                    </h3>
                    <span className="text-xs text-gray-500">
                      {formatLastMessageTime(room.last_message_at)}
                    </span>
                  </div>

                  <div className="flex items-center justify-between">
                    <div className="text-sm text-gray-500 truncate flex-1 mr-2">
                      <span>{room.last_message_content}</span>
                      {/* Show item info if available */}
                      {room.item_name && (
                        <span className="text-xs text-gray-400 block">
                          {room.item_name}
                          {room.item_price && ` - ${room.item_price}`}
                        </span>
                      )}
                    </div>

                    {room.unread_count > 0 && (
                      <span className="inline-flex items-center justify-center px-2 py-1 text-xs font-bold leading-none text-white bg-red-500 rounded-full flex-shrink-0">
                        {room.unread_count}
                      </span>
                    )}
                  </div>
                </div>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
};

export default ConversationList;
