export type BodyDeserialized = {
  model: string;
  messages: Array<{
    role: 'system' | 'user' | 'assistant' | 'tool';
    content: string;
  }>;
};

export type RequestOptions = {
  method?: 'POST' | 'GET' | 'PUT' | 'DELETE';
  headers?: HeadersInit;
  body?: string;
  path: string;
  signal?: AbortSignal;
  timeout?: number;
  requestKey?: string;
};

export type ChatConversation = {
  id?: number;
  userId: string;
  title?: string;
  model?: string;
  createdAt?: Date;
  updatedAt?: Date;
}

export type ChatMessage = {
  role: 'system' | 'user' | 'assistant' | 'tool';
  content: string;
  images?: string[];
  tool_calls?: unknown[];
  model?: string;
};

export type ChatRequest = {
  model: string;
  messages: ChatMessage[];
  tools?: Record<string, unknown>;
  format?: string;
  options?: string[];
  stream?: boolean;
  keep_alive?: string;
}

export type ChatResponse = {
  done: boolean;
  message?: ChatMessage;
  createdAt: string;
  model: string;
  context?: number[];
  done_reason?: string;
  total_duration?: number;
  load_duration?: number;
  prompt_eval_count?: number;
  prompt_eval_duration?: number;
  eval_count?: number;
  eval_duration?: number;
};

export type ModelDetails = {
  parentModel: string;
  format: string;
  family: string;
  families: string[];
  parameterSize: number;
  quantizationLevel: string;
}

export type Model = {
  name: string;
  modifiedAt: Date;
  size: number;
  digest: string;
  details: ModelDetails;
}
