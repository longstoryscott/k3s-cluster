export interface User {
    id: string;
    name: string;
    email: string;
    accessToken?: string;
    refreshToken?: string;
}

export interface ChatMessage {
    id: string;
    content: string;
    role: 'user' | 'assistant';
    timestamp: string;
}

export interface ChatHistory {
    messages: ChatMessage[];
}

export interface ApiResponse<T> {
    success: boolean;
    data: T;
    error?: string;
}

export interface ApiError {
    message: string;
    code: number;
}

export interface Model {
    id: string;
    name: string;
    description: string;
    version: string;
}

export interface Settings {
    apiBaseUrl: string;
    theme: 'light' | 'dark';
}