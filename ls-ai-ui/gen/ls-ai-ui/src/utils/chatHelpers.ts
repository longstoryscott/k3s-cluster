import { ChatResponse } from '../../../../src/api/types';

export const formatChatResponse = (response: ChatResponse): string => {
  if (response.message) {
    return `${response.message.role}: ${response.message.content}`;
  }
  return 'No response available';
};

export const isResponseDone = (response: ChatResponse): boolean => {
  return response.done;
};

export const extractContext = (response: ChatResponse): string => {
  return response.context ? response.context.join(', ') : '';
};

export const parseMarkdown = (markdown: string): string => {
  // Placeholder for markdown parsing logic
  return markdown; // This should be replaced with actual parsing logic
};