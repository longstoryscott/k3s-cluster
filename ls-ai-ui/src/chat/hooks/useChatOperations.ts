import { useCallback, useRef, useEffect } from 'react';
import { ChatState, ChatActions } from './useChatState';
import { useAuth } from '../../auth';
import { chat, getManyConversations, getMessages, removeConversation, startConversation, updateConversationTitle, ChatUserMessage, getModels, getToken } from '../../api';
import { useWebSearch } from './useWebSearch';

export const useChatOperations = (state: ChatState, actions: ChatActions) => {
  const auth = useAuth();
  const debounceTimers = useRef<Record<string, ReturnType<typeof setTimeout>>>({});
  const abortController = useRef<AbortController | null>(null);
  const {
    detectWebSearchIntent,
    performWebSearch,
    formatSearchResultsForLLM,
    isWebSearchEnabled,
    isSearching
  } = useWebSearch({ autoSearch: true });

  // Sync isSearching state from useWebSearch hook to ChatState
  useEffect(() => {
    actions.setIsSearching(isSearching);
  }, [isSearching, actions]);

  // Fetch models
  const fetchModels = useCallback(async () => {
    actions.setIsLoading(true);
    actions.setError(null);
    try {
      const modelsData = await getModels(getToken(auth.user));
      actions.setModels(modelsData?.models);
    } catch (err: unknown) {
      actions.setError((err as Error).message);
      console.error("Error fetching models:", err);
    } finally {
      actions.setIsLoading(false);
    }
  }, [actions, auth.user]);

  // Fetch conversations
  const fetchConversations = useCallback(async () => {
    actions.setIsLoading(true);
    actions.setError(null);

    try {
      const conversationsData = await getManyConversations(getToken(auth.user));
      actions.setConversations(conversationsData);
    } catch (err: unknown) {
      actions.setError((err as Error).message);
      console.error("Error fetching conversations:", err);
    } finally {
      actions.setIsLoading(false);
    }
  }, [actions, auth.user]);

  // Fetch messages for a specific conversation
  const fetchMessages = useCallback(async (conversationId: number) => {
    actions.setIsLoading(true);
    actions.setError(null);
    // Clear the response state to avoid showing stale data
    actions.setResponse('');

    try {
      const fetchedMessages = await getMessages(getToken(auth.user), conversationId);
      actions.setMessages(msgs => [...(msgs ?? []), ...(fetchedMessages ?? []).filter(m => !msgs.find(msg => msg.id === m.id))]);
      // Find and set the current conversation
      const conversation = state.conversations.find(c => c.id === conversationId);
      if (conversation) {
        actions.setCurrentConversation(conversation);
      } else {
        // If not in our list, fetch all conversations
        const conversationsData = await getManyConversations(getToken(auth.user));
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
  }, [actions, auth.user, state.conversations]);

  // Start a new conversation
  const startNewConversation = useCallback(async (model?: string) => {
    actions.setIsLoading(true);
    actions.setError(null);

    const modelToUse = model || state.selectedModel;

    try {
      const newConversation = await startConversation(getToken(auth.user), modelToUse);

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
  }, [state.selectedModel, actions, auth.user]);

  // Reset response
  const resetResponse = useCallback(() => {
    actions.setResponse('');
  }, [actions]);

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

      // Check if we should do a web search based on the prompt
      let finalMessage = { ...message };

      if (isWebSearchEnabled) {
        try {
          const needsSearch = await detectWebSearchIntent(message.content);

          if (needsSearch) {
            actions.setResponse('Searching the web for information...');
            // We don't need to explicitly set isSearching here as it's handled by the useWebSearch hook
            // and synced via the useEffect above

            const searchResults = await performWebSearch(message.content);
            if (searchResults && !searchResults.error) {
              // Format the search results for the LLM
              const formattedResults = formatSearchResultsForLLM(searchResults);

              // Add search results to the prompt
              finalMessage = {
                ...message,
                content: `${formattedResults}\n\nBased on the above web search results, please respond to: ${message.content}`
              };

              // Add a system note about the search
              actions.setResponse('Web search complete. Processing your request...');
            }
          }
        } catch (searchError) {
          console.error('Error during web search:', searchError);
          // Continue with original message if search fails
        }
      }

      // Update UI immediately with the user message
      actions.addMessage(messageWithStatus);

      // Stream the assistant's response
      for await (const chunk of chat(getToken(auth.user), state.messages, finalMessage)) {
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
  }, [
    state.isTyping,
    state.currentConversation,
    state.messages,
    fetchMessages,
    auth.user,
    startNewConversation,
    actions,
    isWebSearchEnabled,
    detectWebSearchIntent,
    performWebSearch,
    formatSearchResultsForLLM
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
      await removeConversation(getToken(auth.user), id);

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
  }, [state.isLoading, state.currentConversation, actions, auth.user]);

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
        await updateConversationTitle(getToken(auth.user), id, title);
      } catch (err: unknown) {
        actions.setError((err as Error).message);
        console.error("Error updating conversation title:", err);
      }
      delete debounceTimers.current['updateTitle'];
    }, 500);
  }, [auth.user, actions]);

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
    fetchModels,
    response: state.response,
    isTyping: state.isTyping,
    isSearching, // Expose isSearching to components using this hook
    resetResponse,
    abortGeneration: useCallback(() => {
      if (abortController.current) {
        abortController.current.abort();
        abortController.current = null;
        actions.setIsTyping(false);
      }
    }, [actions])
  };
};