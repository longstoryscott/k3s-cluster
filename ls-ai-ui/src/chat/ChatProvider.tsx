import React, { useEffect, useRef } from 'react';
import { ChatContext } from './useChat';
import { useAuth } from '../auth';
import { useChatState } from './hooks/useChatState';
import { useChatOperations } from './hooks/useChatOperations';
import { Model } from '../api';

export interface ChatContextType {
  // State
  messages: ReturnType<typeof useChatState>[0]['messages'];
  conversations: ReturnType<typeof useChatState>[0]['conversations'];
  currentConversation: ReturnType<typeof useChatState>[0]['currentConversation'];
  isLoading: boolean;
  error: string | null;
  isTyping: boolean;
  response: string;
  selectedModel: string;
  models: Model[];
  
  // Actions
  sendMessage: ReturnType<typeof useChatOperations>['sendMessage'];
  retryMessage: ReturnType<typeof useChatOperations>['retryMessage'];
  fetchMessages: ReturnType<typeof useChatOperations>['fetchMessages'];
  fetchConversations: ReturnType<typeof useChatOperations>['fetchConversations'];
  deleteConversation: ReturnType<typeof useChatOperations>['deleteConversation'];
  startNewConversation: ReturnType<typeof useChatOperations>['startNewConversation'];
  selectConversation: ReturnType<typeof useChatOperations>['selectConversation'];
  setConversationTitle: ReturnType<typeof useChatOperations>['setConversationTitle'];
  setSelectedModel: ReturnType<typeof useChatState>[1]['setSelectedModel'];
  fetchModels: ReturnType<typeof useChatOperations>['fetchModels'];
  setCurrentConversation: ReturnType<typeof useChatState>[1]['setCurrentConversation'];
}

export const ChatProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const auth = useAuth();
  
  // Use our custom hooks
  const [state, actions] = useChatState();
  const operations = useChatOperations(state, actions);
  
  // Track API request to prevent duplicates
  const apiRequestInProgress = useRef(false);
  const isFirstLoad = useRef(true);

  // Load conversations on first mount
  useEffect(() => {
    if (auth.isAuthenticated && isFirstLoad.current && !apiRequestInProgress.current) {
      isFirstLoad.current = false;
      apiRequestInProgress.current = true;
      (async () => {
        await operations.fetchModels();
        await operations.fetchConversations();
        apiRequestInProgress.current = false;
      })();
    }
  }, [auth.isAuthenticated, operations]);

  // Construct the context value from our hooks
  const contextValue: ChatContextType = {
    // State
    messages: state.messages,
    conversations: state.conversations,
    currentConversation: state.currentConversation,
    isLoading: state.isLoading,
    error: state.error,
    isTyping: state.isTyping,
    response: state.response,
    selectedModel: state.selectedModel,
    models: state.models,
    
    // Actions
    sendMessage: operations.sendMessage,
    retryMessage: operations.retryMessage,
    fetchMessages: operations.fetchMessages,
    fetchConversations: operations.fetchConversations,
    deleteConversation: operations.deleteConversation,
    startNewConversation: operations.startNewConversation,
    selectConversation: operations.selectConversation,
    setConversationTitle: operations.setConversationTitle,
    setSelectedModel: actions.setSelectedModel,
    setCurrentConversation: actions.setCurrentConversation,
    fetchModels: operations.fetchModels
  };

  return <ChatContext.Provider value={contextValue}>{children}</ChatContext.Provider>;
};
