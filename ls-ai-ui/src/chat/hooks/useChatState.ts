import { useState, useCallback } from 'react';
import { ChatConversation, Model, ChatMessage } from '../../api/types';

export interface ChatState {
  messages: ChatMessage[];
  conversations: ChatConversation[];
  currentConversation: ChatConversation | null;
  isLoading: boolean;
  error: string | null;
  isTyping: boolean;
  response: string;
  selectedModel: string;
  models: Model[];
}

export interface ChatActions {
  setMessages: React.Dispatch<React.SetStateAction<ChatMessage[]>>;
  setConversations: (conversations: ChatConversation[]) => void;
  setCurrentConversation: (conversation: ChatConversation | null) => void;
  setIsLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
  setIsTyping: (typing: boolean) => void;
  setResponse: React.Dispatch<React.SetStateAction<string>>;
  setSelectedModel: (model: string) => void;
  addMessage: (message: ChatMessage) => void;
  addConversation: (conversation: ChatConversation) => void;
  updateConversationInList: (id: number, updates: Partial<ChatConversation>) => void;
  removeConversationFromList: (id: number) => void;
  setModels: (models: Model[]) => void;
}

export const useChatState = (): [ChatState, ChatActions] => {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [conversations, setConversations] = useState<ChatConversation[]>([]);
  const [currentConversation, setCurrentConversation] = useState<ChatConversation | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isTyping, setIsTyping] = useState(false);
  const [response, setResponse] = useState<string>('');
  const [selectedModel, setSelectedModelState] = useState<string>(() => {
    return localStorage.getItem('selectedModel') || '';
  });
  const [models, setModelsState] = useState<Model[]>([]);

  const setModels = useCallback((models: Model[]) => {
    setModelsState(models);
  }, []);

  const setSelectedModel = useCallback((model: string) => {
    setSelectedModelState(model);
    localStorage.setItem('selectedModel', model);
  }, []);

  const addMessage = useCallback((message: ChatMessage) => {
    setMessages(prev => [...prev, message]);
  }, []);

  const addConversation = useCallback((conversation: ChatConversation) => {
    setConversations(prev => [conversation, ...(prev ?? [])]);
  }, []);

  const updateConversationInList = useCallback((id: number, updates: Partial<ChatConversation>) => {
    setConversations(prev =>
      prev.map(c => c.id === id ? { ...c, ...updates } : c)
    );

    setCurrentConversation(prev =>
      prev?.id === id ? { ...prev, ...updates } : prev
    );
  }, []);

  const removeConversationFromList = useCallback((id: number) => {
    setConversations(prev => prev.filter(c => c.id !== id));

    setCurrentConversation(prev =>
      prev?.id === id ? null : prev
    );
  }, []);

  const state: ChatState = {
    messages,
    conversations,
    currentConversation,
    isLoading,
    error,
    isTyping,
    response,
    selectedModel,
    models
  };

  const actions: ChatActions = {
    setMessages,
    setConversations,
    setCurrentConversation,
    setIsLoading,
    setError,
    setIsTyping,
    setResponse,
    setSelectedModel,
    addMessage,
    addConversation,
    updateConversationInList,
    removeConversationFromList,
    setModels
  };

  return [state, actions];
};