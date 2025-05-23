import { useCallback, useRef } from 'react';
import { ChatState, ChatActions } from './useChatState';
import { useAuth } from '../../auth';
import { chat, getManyConversations, getMessages, removeConversation, startConversation, updateConversationTitle, ChatUserMessage, getModels } from '../../api';

export const useChatOperations = (state: ChatState, actions: ChatActions) => {
  const auth = useAuth();
  const debounceTimers = useRef<Record<string, ReturnType<typeof setTimeout>>>({});

  // Helper function to get access token
  const getToken = useCallback(() => {
    return auth.user?.accessToken || '';
  }, [auth.user?.accessToken]);

  // Fetch models
  const fetchModels = useCallback(async () => {
    actions.setIsLoading(true);
    actions.setError(null);
    try {
      const modelsData = await getModels(getToken());
      actions.setModels(modelsData?.models);
    } catch (err: unknown) {
      actions.setError((err as Error).message);
      console.error("Error fetching models:", err);
    } finally {
      actions.setIsLoading(false);
    }
  }, [actions, getToken]);

  // Fetch conversations
  const fetchConversations = useCallback(async () => {
    actions.setIsLoading(true);
    actions.setError(null);

    try {
      const conversationsData = await getManyConversations(getToken());
      actions.setConversations(conversationsData);
    } catch (err: unknown) {
      actions.setError((err as Error).message);
      console.error("Error fetching conversations:", err);
    } finally {
      actions.setIsLoading(false);
    }
  }, [actions, getToken]);

  // Fetch messages for a specific conversation
  const fetchMessages = useCallback(async (conversationId: number) => {
    actions.setIsLoading(true);
    actions.setError(null);
    // Clear the response state to avoid showing stale data
    actions.setResponse('');

    try {
      const fetchedMessages = await getMessages(getToken(), conversationId);
      actions.setMessages(msgs => [...(msgs ?? []), ...(fetchedMessages ?? []).filter(m => !msgs.find(msg => msg.id === m.id))]);
      // Find and set the current conversation
      const conversation = state.conversations.find(c => c.id === conversationId);
      if (conversation) {
        actions.setCurrentConversation(conversation);
      } else {
        // If not in our list, fetch all conversations
        const conversationsData = await getManyConversations(getToken());
        // Update the full conversations list
        actions.setConversations(conversationsData);

        // Find and set the current conversation from the fetched data
        const foundConversation = conversationsData.find(c => c.id === conversationId);
        if (foundConversation) {
          actions.setCurrentConversation(foundConversation);
        }
      }
    } catch (err: unknown) {
      actions.setError((err as Error).message);
      console.error("Error fetching messages:", err);
    } finally {
      actions.setIsLoading(false);
    }
  }, [actions, getToken, state.conversations]);

  // Start a new conversation
  const startNewConversation = useCallback(async (model?: string) => {
    actions.setIsLoading(true);
    actions.setError(null);

    const modelToUse = model || state.selectedModel;

    try {
      const newConversation = await startConversation(getToken(), modelToUse);

      // Update local state
      actions.setCurrentConversation(newConversation);
      actions.setMessages([]);
      actions.setResponse('');

      // Add to conversations list
      actions.addConversation(newConversation);

      return newConversation.id ?? -1;
    } catch (err: unknown) {
      actions.setError((err as Error).message);
      console.error("Error creating conversation:", err);
      throw err;
    } finally {
      actions.setIsLoading(false);
    }
  }, [state.selectedModel, actions, getToken]);

  // Send a message in the current conversation
  const sendMessage = useCallback(async (message: ChatUserMessage) => {
    if (state.isTyping) {
      console.warn("Already typing, please wait.");
      return;
    }

    actions.setIsLoading(true);
    actions.setError(null);
    actions.setIsTyping(true);

    // Create a unique ID for this message instance to help with tracking
    const messageWithStatus = {
      ...message,
      status: 'sending' as const
    };

    try {
      // Make sure we have a conversation
      let conversationId = state.currentConversation?.id;
      if (!conversationId) {
        conversationId = await startNewConversation();
      }
      await fetchMessages(conversationId ?? -1);
      actions.setResponse('');

      // Update UI immediately with the user message
      actions.addMessage(messageWithStatus);

      // Stream the assistant's response
      for await (const chunk of chat(getToken(), state.messages, message)) {
        // Use functional update to ensure we're always working with the latest state
        actions.setResponse(r => r + chunk);
      }

    } catch (err: unknown) {
      console.error("Error sending message:", err);
      actions.setError((err as Error).message);
    } finally {
      actions.setIsLoading(false);
      actions.setIsTyping(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [
    state.isTyping,
    state.currentConversation,
    state.messages,
    fetchMessages,
    getToken,
    startNewConversation
  ]);

  // Retry a failed message
  const retryMessage = useCallback(async (failedMessage: ChatUserMessage) => {
    if (state.isTyping) {
      console.warn("Already processing a message, please wait.");
      return;
    }

    // Find the message in the current state that needs to be retried
    const messageToRetry = state.messages.find(msg =>
      msg.role === failedMessage.role &&
      msg.content === failedMessage.content &&
      msg.status === 'error'
    );

    if (!messageToRetry) {
      console.warn("Failed message not found in current state.");
      return;
    }

    // Create a new message object from the failed one, without the error status
    const newMessage: ChatUserMessage = {
      role: failedMessage.role,
      content: failedMessage.content,
      conversationId: failedMessage.conversationId
      // Don't include the error status
    };

    // Send the message again
    await sendMessage(newMessage);
  }, [state.isTyping, state.messages, sendMessage]);

  // Delete a conversation
  const deleteConversation = useCallback(async (id: number) => {
    if (state.isLoading) {
      return;
    }

    actions.setIsLoading(true);
    actions.setError(null);

    try {
      await removeConversation(getToken(), id);

      // Update local state
      actions.removeConversationFromList(id);

      // If this was the current conversation, clear it
      if (state.currentConversation?.id === id) {
        actions.setMessages([]);
        actions.setResponse('');
      }
    } catch (err: unknown) {
      actions.setError((err as Error).message);
      console.error("Error deleting conversation:", err);
    } finally {
      actions.setIsLoading(false);
    }
  }, [state.isLoading, state.currentConversation, actions, getToken]);

  // Select an existing conversation
  const selectConversation = useCallback(async (id: number) => {
    actions.setIsLoading(true);
    actions.setError(null);
    actions.setMessages([]);

    try {
      await fetchMessages(id);
    } catch (err: unknown) {
      actions.setError((err as Error).message);
      console.error("Error selecting conversation:", err);
    } finally {
      actions.setIsLoading(false);
    }
  }, [actions, fetchMessages]);

  // Debounced function to update conversation title
  const debouncedUpdateTitle = useCallback((id: number, title: string) => {
    if (debounceTimers.current['updateTitle']) {
      clearTimeout(debounceTimers.current['updateTitle']);
    }

    debounceTimers.current['updateTitle'] = setTimeout(async () => {
      try {
        await updateConversationTitle(getToken(), id, title);
      } catch (err: unknown) {
        actions.setError((err as Error).message);
        console.error("Error updating conversation title:", err);
      }
      delete debounceTimers.current['updateTitle'];
    }, 500);
  }, [getToken, actions]);

  // Update conversation title
  const setConversationTitle = useCallback(async (id: number, title: string) => {
    actions.updateConversationInList(id, { title });
    debouncedUpdateTitle(id, title);
  }, [actions, debouncedUpdateTitle]);

  return {
    fetchConversations,
    fetchMessages,
    startNewConversation,
    sendMessage,
    retryMessage,
    deleteConversation,
    selectConversation,
    setConversationTitle,
    fetchModels
  };
};