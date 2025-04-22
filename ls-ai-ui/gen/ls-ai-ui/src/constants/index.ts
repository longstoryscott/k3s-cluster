export const API_ENDPOINTS = {
  CHAT: 'api/chat',
  LOGIN: 'api/login',
  LOGOUT: 'api/logout',
  REFRESH_TOKEN: 'api/token/refresh'
};

export const DEFAULT_MODEL = 'gemma3:1b';

export const MAX_CHAT_HISTORY = 100;

export const ERROR_MESSAGES = {
  LOGIN_FAILED: 'Login failed. Please check your credentials.',
  CHAT_ERROR: 'An error occurred while sending the message. Please try again.',
  TOKEN_REFRESH_FAILED: 'Session expired. Please log in again.'
};