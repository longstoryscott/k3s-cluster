import { gen, getHeaders, req } from "./base";
import { ChatMessage, ChatUserMessage } from "./types";

export async function* chat(accessToken: string, messages: ChatMessage[], message: ChatUserMessage) {

  try {
    const generator = gen({
      body: JSON.stringify({
        messages: [...messages, message],
        conversationId: message.conversationId
      }),
      method: 'POST',
      headers: getHeaders(accessToken),
      path: 'api/chat'
    });

    for await (const chunk of generator) {
      yield chunk.message?.content;

      if (chunk.done) {
        break;
      }
    }
  } catch (error) {
    console.error('Chat API error:', error);
    throw error;
  }
};

export const getMessages = async (accessToken: string, conversationId: number) =>
  req<ChatUserMessage[]>({
    method: 'GET',
    headers: getHeaders(accessToken),
    path: `api/conversations/${conversationId}/messages`
  });

