import { gen, getHeaders, req } from "./base";
import { ChatMessage, ChatUserMessage } from "./types";

export const chat = (accessToken: string, model: string, messages: ChatMessage[], message: ChatUserMessage) => {
  try {
    return gen({
      body: JSON.stringify({
        model: model || 'phi3.5',
        messages: [...messages, message],
        conversationId: message.conversationId
      }),
      method: 'POST',
      headers: getHeaders(accessToken),
      path: 'api/chat'
    });
  } catch (error) {
    // Ensure we return a structured error that can be handled properly
    console.error('Chat API error:', error);
    throw error; // Re-throw to allow proper handling up the chain
  }
};

export const getMessages = async (accessToken: string, conversationId: number) =>
  req<ChatUserMessage[]>({
    method: 'GET',
    headers: getHeaders(accessToken),
    path: `api/conversations/${conversationId}/messages`
  });

