import React, { useState } from 'react';
import { Channel } from '../types';

interface AIAssistantProps {
  channel: Channel;
}

interface Suggestion {
  type: string;
  confidence: number;
  content: string;
}

const AIAssistant: React.FC<AIAssistantProps> = ({ channel }) => {
  const [isExpanded, setIsExpanded] = useState(true);

  // Mock AI suggestions - in real app this would come from your AI service
  const suggestions: Suggestion[] = [
    {
      type: 'Response',
      confidence: 98,
      content: "Great! I'm heading out now. Here's my direct number: (555) 123-4567. Text me if anything changes before 4:30pm.",
    },
    {
      type: 'Revenue Opportunity',
      confidence: 87,
      content: 'Emergency job with potential for additional work. Customer concerned about costs but willing to pay for quality service.',
    },
    {
      type: 'Service Reminder',
      confidence: 93,
      content: 'Send arrival notification when 15 minutes away, plus follow-up satisfaction survey tomorrow.',
    },
  ];

  const getConfidenceColor = (confidence: number) => {
    if (confidence >= 90) return 'text-green-600';
    if (confidence >= 70) return 'text-orange-500';
    return 'text-red-500';
  };

  const getConfidenceLabel = (confidence: number) => {
    if (confidence >= 90) return 'confident';
    if (confidence >= 70) return 'confident';
    return 'confident';
  };

  if (!isExpanded) {
    return (
      <div className="w-12 border-l border-gray-200 bg-white flex flex-col items-center p-2">
        <button
          onClick={() => setIsExpanded(true)}
          className="p-2 text-blue-600 hover:bg-blue-50 rounded-lg"
        >
          <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
          </svg>
        </button>
        <div className="mt-4 text-blue-600 transform -rotate-90 text-sm font-medium whitespace-nowrap">
          AI Assistant
        </div>
      </div>
    );
  }

  return (
    <div className="w-80 border-l border-gray-200 bg-white flex flex-col">
      {/* Header */}
      <div className="p-4 border-b border-gray-200">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-2">
            <div className="w-6 h-6 bg-blue-600 rounded-full flex items-center justify-center">
              <svg className="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
            </div>
            <h3 className="text-lg font-semibold text-gray-900">AI Assistant</h3>
          </div>
          <button
            onClick={() => setIsExpanded(false)}
            className="p-1 text-gray-400 hover:text-gray-600"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
        <p className="text-sm text-gray-600 mt-1">
          AI suggestions for active conversation
        </p>
      </div>

      {/* Suggestions */}
      <div className="flex-1 overflow-y-auto p-4">
        <div className="space-y-4">
          <div className="bg-blue-50 border border-blue-200 rounded-lg p-3">
            <div className="text-sm text-blue-800 font-medium mb-1">Suggested</div>
            <div className="flex items-center justify-between mb-2">
              <span className="text-sm font-medium text-gray-900">Response</span>
              <span className="text-xs text-blue-600 font-medium">for Mike Rodriguez</span>
            </div>
            <div className="text-sm text-gray-700 mb-3">
              Great! I'm heading out now. Here's my direct number: (555) 123-4567.
              Text me if anything changes before 4:30pm.
            </div>
            <div className="flex space-x-2">
              <button className="btn-primary text-xs py-1 px-3">
                Send
              </button>
              <button className="btn-secondary text-xs py-1 px-3">
                Dismiss
              </button>
            </div>
          </div>

          {suggestions.slice(1).map((suggestion, index) => (
            <div key={index} className="bg-gray-50 border border-gray-200 rounded-lg p-3">
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm font-medium text-gray-900">
                  {suggestion.type}
                </span>
                <div className="flex items-center space-x-1">
                  <span className={`text-xs font-medium ${getConfidenceColor(suggestion.confidence)}`}>
                    {suggestion.confidence}%
                  </span>
                  <span className="text-xs text-gray-500">
                    {getConfidenceLabel(suggestion.confidence)}
                  </span>
                </div>
              </div>
              <div className="text-sm text-gray-700 mb-3">
                {suggestion.content}
              </div>
              <div className="flex space-x-2">
                {suggestion.type === 'Response' ? (
                  <>
                    <button className="btn-primary text-xs py-1 px-3">
                      Send
                    </button>
                    <button className="btn-secondary text-xs py-1 px-3">
                      Dismiss
                    </button>
                  </>
                ) : suggestion.type === 'Service Reminder' ? (
                  <>
                    <button className="bg-orange-500 hover:bg-orange-600 text-white text-xs py-1 px-3 rounded">
                      Setup
                    </button>
                    <button className="btn-secondary text-xs py-1 px-3">
                      Dismiss
                    </button>
                  </>
                ) : (
                  <>
                    <button className="bg-green-500 hover:bg-green-600 text-white text-xs py-1 px-3 rounded">
                      Note
                    </button>
                    <button className="btn-secondary text-xs py-1 px-3">
                      Dismiss
                    </button>
                  </>
                )}
              </div>
            </div>
          ))}
        </div>

        {/* Quick Reply Section */}
        <div className="mt-6 pt-4 border-t border-gray-200">
          <div className="text-sm font-medium text-gray-900 mb-2">
            92% Quick Reply
          </div>
          <div className="text-xs text-gray-600 mb-3">
            AI suggestions for faster responses
          </div>
          <div className="space-y-2">
            <button className="w-full text-left p-2 text-sm text-gray-700 hover:bg-gray-50 rounded border border-gray-200">
              I understand the concern. If it's a simple drain snake job (under 30 mins)...
            </button>
            <button className="w-full text-left p-2 text-sm text-gray-700 hover:bg-gray-50 rounded border border-gray-200">
              That sounds reasonable. How soon can you be here?
            </button>
            <button className="w-full text-left p-2 text-sm text-gray-700 hover:bg-gray-50 rounded border border-gray-200">
              I can be there by 4:30pm today. What's your address?
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default AIAssistant;