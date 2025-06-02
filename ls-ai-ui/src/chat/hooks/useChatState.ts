import { useState, useCallback, useMemo } from 'react';
import { ChatConversation, Model, ChatMessage } from '../../api/types';
import { useAuth } from '../../auth';

export interface ChatState {
  messages: ChatMessage[];
  conversations: { [key: string]: ChatConversation[] };
  currentConversation: ChatConversation | null;
  isLoading: boolean;
  error: string | null;
  isTyping: boolean;
  response: string;
  selectedModel: string;
  models: Model[];
  isSearching: boolean; // Add isSearching property
}

export interface ChatActions {
  setMessages: React.Dispatch<React.SetStateAction<ChatMessage[]>>;
  setConversations: React.Dispatch<React.SetStateAction<{ [key: string]: ChatConversation[] }>>;
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
  setIsSearching: (searching: boolean) => void; // Add setIsSearching action
}

export const useChatState = (): [ChatState, ChatActions] => {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [conversations, setConversations] = useState<{ [key: string]: ChatConversation[] }>({});
  const [currentConversation, setCurrentConversation] = useState<ChatConversation | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isTyping, setIsTyping] = useState(false);
  const [response, setResponse] = useState<string>('');
  const [selectedModel, setSelectedModelState] = useState<string>(() => {
    return localStorage.getItem('selectedModel') || '';
  });
  const [models, setModelsState] = useState<Model[]>([]);
  const [isSearching, setIsSearching] = useState(false); // Initialize isSearching state
  const { user } = useAuth(); // Assuming useAuth is a custom hook to get user info
  const currentUserId = useMemo(() => user?.profile?.preferred_username ?? '', [user]);

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
    if (!currentUserId) {
      return;
    }
    setConversations(prev => ({
      ...prev,
      [currentUserId]: [conversation, ...(prev[currentUserId] || [])]
    }));
  }, [currentUserId]);

  const updateConversationInList = useCallback((id: number, updates: Partial<ChatConversation>) => {
    if (!currentUserId) {
      return;
    }
    setConversations(prev => ({
      ...prev,
      [currentUserId]: prev[currentUserId].map(c => c.id === id ? { ...c, ...updates } : c)
    }));

    setCurrentConversation(prev =>
      prev?.id === id ? { ...prev, ...updates } : prev
    );
  }, [currentUserId]);

  const removeConversationFromList = useCallback((id: number) => {
    if (!currentUserId) {
      return;
    }

    setConversations(prev => ({
      ...prev,
      [currentUserId]: prev[currentUserId].filter(c => c.id !== id)
    }));


    setCurrentConversation(prev =>
      prev?.id === id ? null : prev
    );
  }, [currentUserId]);

  const state: ChatState = {
    messages,
    conversations,
    currentConversation,
    isLoading,
    error,
    isTyping,
    response,
    selectedModel,
    models,
    isSearching // Include isSearching in the state
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
    setModels,
    setIsSearching // Include setIsSearching in the actions
  };

  return [state, actions];
};