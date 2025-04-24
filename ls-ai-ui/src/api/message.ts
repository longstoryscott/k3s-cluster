import { gen, getHeaders, req } from "./base";
import { ChatMessage } from "./types";

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

