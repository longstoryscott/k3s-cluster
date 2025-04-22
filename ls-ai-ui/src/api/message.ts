import { gen, req } from "./base";
import { ChatMessage } from "./types";

// Headers for API requests
const getHeaders = (accessToken: string) => ({
  Authorization: `Bearer ${accessToken}`,
  'Content-Type': 'application/json'
});

export const chat = (accessToken: string, model: string, messages: ChatMessage[], message: ChatMessage) => gen({
  body: JSON.stringify({
    model: model || 'phi3.5',
    messages: [...messages, message]
  }),
  method: 'POST',
  headers: getHeaders(accessToken),
  path: 'api/chat'
});

export const getMessages = async (accessToken: string, conversationId: number) =>
  req<ChatMessage[]>({
    method: 'GET',
    headers: getHeaders(accessToken),
    path: `api/conversations/${conversationId}/messages`
  });

