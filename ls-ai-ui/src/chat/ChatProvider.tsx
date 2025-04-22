import React, { useState, useEffect, useRef, useCallback } from 'react';
import { ChatContext } from './useChat';
import { useAuth } from '../auth';
import { ChatMessage, ChatConversation } from '../api/types';
import { getMessages, chat } from '../api/message';
import { getManyConversations, startConversation, removeConversation, updateConversationTitle } from '../api/conversation';

export interface ChatContextType {
  // State
  messages: ChatMessage[];
  conversations: ChatConversation[];
  currentConversation: ChatConversation | null;
  isLoading: boolean;
  error: string | null;
  isTyping: boolean;
  response: string;
  selectedModel: string;
  
  // Actions
  sendMessage: (message: ChatMessage) => Promise<void>;
  fetchMessages: (conversationId: number) => Promise<void>;
  fetchConversations: () => Promise<void>;
  deleteConversation: (id: number) => Promise<void>;
  startNewConversation: (model?: string) => Promise<number>;
  selectConversation: (id: number) => Promise<void>;
  handleTyping: (typing: boolean) => void;
  setConversationTitle: (id: number, title: string) => Promise<void>;
  setSelectedModel: (model: string) => void;
}

export const ChatProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const auth = useAuth();
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [conversations, setConversations] = useState<ChatConversation[]>([]);
  const [currentConversation, setCurrentConversation] = useState<ChatConversation | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isTyping, setIsTyping] = useState(false);
  const [response, setResponse] = useState<string>('');
  const [selectedModel, setSelectedModelState] = useState<string>(() => {
    return localStorage.getItem('selectedModel') || 'llama3-8b';
  });

  // Create debounced functions
  const debouncedSetTyping = useCallback((typing: boolean) => {
    setIsTyping(typing);
  }, []);

  // Set selected model and persist to localStorage
  const setSelectedModel = useCallback((model: string) => {
    setSelectedModelState(model);
    localStorage.setItem('selectedModel', model);
  }, []);

  // Debounced function to update conversation title
  const debouncedUpdateTitle = useCallback(async (accessToken: string, id: number, title: string) => {
    try {
      await updateConversationTitle(accessToken, id, title);
      setConversations(prev => prev.map(c => 
        c.id === id ? { ...c, title } : c
      ));
      
      if (currentConversation?.id === id) {
        setCurrentConversation(prev => prev ? { ...prev, title } : null);
      }
    } catch (err: unknown) {
      setError((err as Error).message);
      console.error("Error updating conversation title:", err);
    } finally {
      setIsLoading(false);
    }
  }, [currentConversation]);

  // Fetch messages for a specific conversation
  const fetchMessages = useCallback(async (conversationId: number) => {
    if (isLoading) {
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      const fetchedMessages = await getMessages(auth.user?.accessToken || '', conversationId);
      setMessages(fetchedMessages || []);
      
      // Find and set the current conversation
      const conversation = conversations.find(c => c.id === conversationId);
      if (conversation) {
        setCurrentConversation(conversation);
      } else {
        // If not in our list, fetch the conversation details
        const conversationsData = await getManyConversations(auth.user?.accessToken || '');
        const foundConversation = conversationsData.find(c => c.id === conversationId);
        if (foundConversation) {
          setCurrentConversation(foundConversation);
          setConversations(prev => [foundConversation, ...prev.filter(c => c.id !== conversationId)]);
        }
      }
    } catch (err: unknown) {
      setError((err as Error).message);
      console.error("Error fetching messages:", err);
    } finally {
      setIsLoading(false);
    }
  }, [isLoading, auth.user?.accessToken, conversations]);

  // Start a new conversation
  const startNewConversation = useCallback(async (model?: string) => {
    if (isLoading) {
      return -1;
    }

    setIsLoading(true);
    setError(null);
    
    const modelToUse = model || selectedModel;

    try {
      const newConversation = await startConversation(auth.user?.accessToken || '', modelToUse);
      
      // Update local state
      setCurrentConversation(newConversation);
      setMessages([]);
      setResponse('');
      
      // Add to conversations list
      setConversations(prev => [newConversation, ...prev]);
      
      return newConversation.id ?? -1;
    } catch (err: unknown) {
      setError((err as Error).message);
      console.error("Error creating conversation:", err);
      throw err;
    } finally {
      setIsLoading(false);
    }
  }, [isLoading, selectedModel, auth.user?.accessToken]);

  // Send a message in the current conversation
  const sendMessage = useCallback(async (message: ChatMessage) => {
    if (isTyping) {
      console.warn("Already typing, please wait.");
      return;
    }

    setIsLoading(true);
    setError(null);
    setIsTyping(true);
    
    try {
      // Make sure we have a conversation
      let conversationId = currentConversation?.id;
      if (!conversationId) {
        conversationId = await startNewConversation();
      }
      
      // Update UI immediately with the user message
      const updatedMessages = [...messages, message];
      setMessages(updatedMessages);
      
      // Clear previous response
      setResponse('');
      
      // Start streaming the assistant's response
      let assistantResponse = '';
      const generator = chat(auth.user?.accessToken || '', selectedModel, messages, message);
      
      for await (const chunk of generator) {
        if (chunk.message?.content) {
          assistantResponse += chunk.message.content;
          setResponse(assistantResponse);
        }
        
        if (chunk.done) {
          // Once done, add full response to our message list
          // This won't be necessary since we'll fetch from server
          break;
        }
      }
      
      // After streaming is complete, refresh messages from the server
      if (conversationId) {
        await fetchMessages(conversationId);
      }
      
    } catch (err: unknown) {
      setError((err as Error).message);
      console.error("Error sending message:", err);
    } finally {
      setIsLoading(false);
      setIsTyping(false);
    }
  }, [isTyping, currentConversation, messages, auth.user?.accessToken, selectedModel, startNewConversation, fetchMessages]);

  // Fetch all conversations
  const fetchConversations = useCallback(async () => {
    if (!auth.isAuthenticated || isLoading) {
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      const conversationsData = await getManyConversations(auth.user?.accessToken || '');
      setConversations(conversationsData);
    } catch (err: unknown) {
      setError((err as Error).message);
      console.error("Error fetching conversations:", err);
    } finally {
      setIsLoading(false);
    }
  }, [isLoading, auth.user?.accessToken, auth.isAuthenticated]);

  // Delete a conversation
  const deleteConversation = useCallback(async (id: number) => {
    if (isLoading) {
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      await removeConversation(auth.user?.accessToken || '', id);
      
      // Update local state
      setConversations(prev => prev.filter(c => c.id !== id));
      
      // If this was the current conversation, clear it
      if (currentConversation?.id === id) {
        setCurrentConversation(null);
        setMessages([]);
      }
    } catch (err: unknown) {
      setError((err as Error).message);
      console.error("Error deleting conversation:", err);
    } finally {
      setIsLoading(false);
    }
  }, [isLoading, auth.user?.accessToken, currentConversation]);

  // Select an existing conversation
  const selectConversation = useCallback(async (id: number) => {
    if (isLoading) {
      return;
    }
    await fetchMessages(id);
  }, [isLoading, fetchMessages]);

  // Update conversation title
  const setConversationTitle = useCallback(async (id: number, title: string) => {
    setIsLoading(true);
    setError(null);
    
    // Update UI immediately for responsiveness
    setConversations(prev => prev.map(c => 
      c.id === id ? { ...c, title } : c
    ));
    
    if (currentConversation?.id === id) {
      setCurrentConversation(prev => prev ? { ...prev, title } : null);
    }
    
    // Use debounced function for the actual API call
    debouncedUpdateTitle(auth.user?.accessToken || '', id, title);
  }, [auth.user?.accessToken, currentConversation, debouncedUpdateTitle]);

  // Handle typing indicator
  const handleTyping = useCallback((typing: boolean) => {
    debouncedSetTyping(typing);
  }, [debouncedSetTyping]);
  
  // Track API request to prevent duplicates
  const apiRequestInProgress = useRef(false);
  const isFirstLoad = useRef(true);

  // Load conversations on first mount
  useEffect(() => {
    if (auth.user && isFirstLoad.current && !apiRequestInProgress.current) {
      isFirstLoad.current = false;
      apiRequestInProgress.current = true;
      fetchConversations().finally(() => {
        apiRequestInProgress.current = false;
      });
    }
  }, [auth.user, fetchConversations]);

  // Context value
  const value: ChatContextType = {
    messages,
    response,
    conversations,
    currentConversation,
    isLoading,
    error,
    isTyping,
    selectedModel,
    sendMessage,
    fetchMessages,
    fetchConversations,
    deleteConversation,
    startNewConversation,
    selectConversation,
    handleTyping,
    setConversationTitle,
    setSelectedModel
  };

  return <ChatContext.Provider value={value}>{children}</ChatContext.Provider>;
};
