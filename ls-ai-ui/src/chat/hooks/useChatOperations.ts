import { useCallback, useRef } from 'react';
import { ChatState, ChatActions } from './useChatState';
import { useAuth } from '../../auth';
import { chat, getManyConversations, getMessages, removeConversation, startConversation, updateConversationTitle, ChatUserMessage, ChatAgentMessage } from '../../api';
import { getModels } from '../../api/model';

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
      actions.setMessages(fetchedMessages || []);

      // Find and set the current conversation
      const conversation = state.conversations.find(c => c.id === conversationId);
      if (conversation) {
        actions.setCurrentConversation(conversation);
      } else {
        // If not in our list, fetch the conversation details
        const conversationsData = await getManyConversations(getToken());
        const foundConversation = conversationsData.find(c => c.id === conversationId);
        if (foundConversation) {
          actions.setCurrentConversation(foundConversation);
          actions.setConversations([
            foundConversation,
            ...state.conversations.filter(c => c.id !== conversationId)
          ]);
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
    // Clear previous response
    actions.setResponse('');

    try {
      // Make sure we have a conversation
      let conversationId = state.currentConversation?.id;
      if (!conversationId) {
        conversationId = await startNewConversation();
      }

      // Update UI immediately with the user message
      actions.addMessage(message);

      // Stream the assistant's response
      const generator = chat(getToken(), state.selectedModel, state.messages, message);
      let completeResponse = ''; // Track the complete response for later addition

      for await (const chunk of generator) {
        if (chunk.message?.content) {
          // Update the streaming response
          completeResponse += chunk.message.content;
          actions.setResponse(completeResponse);
        }

        if (chunk.done) {
          break;
        }
      }

      // When streaming completes, add the complete response to messages
      if (completeResponse) {
        const assistantMessage: ChatAgentMessage = {
          role: 'assistant',
          content: completeResponse
        };
        actions.addMessage(assistantMessage);
      }

      // Keep the final response visible
      actions.setIsTyping(false);

      // Don't reset the response after streaming
      // The response will stay visible until the user starts a new conversation
      // or clicks on a different conversation

    } catch (err: unknown) {
      actions.setError((err as Error).message);
      console.error("Error sending message:", err);
    } finally {
      actions.setIsLoading(false);
      actions.setIsTyping(false);
    }
  }, [
    state.isTyping,
    state.currentConversation,
    state.messages,
    state.selectedModel,
    actions,
    getToken,
    startNewConversation
  ]);

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
    if (state.isLoading) {
      return;
    }
    await fetchMessages(id);
  }, [state.isLoading, fetchMessages]);

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
    deleteConversation,
    selectConversation,
    setConversationTitle,
    fetchModels
  };
};