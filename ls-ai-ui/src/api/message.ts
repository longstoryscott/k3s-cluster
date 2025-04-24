import { gen, getHeaders, req } from "./base";
import { ChatMessage, ChatUserMessage } from "./types";

export const chat = (accessToken: string, model: string, messages: ChatMessage[], message: ChatUserMessage) => {
  console.log(message)
  return gen({
    body: JSON.stringify({
      model: model || 'phi3.5',
      messages: [...messages, message],
      conversationId: message.conversationId
    }),
    method: 'POST',
    headers: getHeaders(accessToken),
    path: 'api/chat'
  })
};

export const getMessages = async (accessToken: string, conversationId: number) =>
  req<ChatUserMessage[]>({
    method: 'GET',
    headers: getHeaders(accessToken),
    path: `api/conversations/${conversationId}/messages`
  });

