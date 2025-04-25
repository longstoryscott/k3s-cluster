import config from "../config";
import { ChatResponse, RequestOptions } from "./types";

export async function* gen(opts: RequestOptions): AsyncGenerator<ChatResponse> {
  opts.headers = {
    ...opts.headers,
    'Content-Type': 'application/json'
  }
  opts.method = opts.method || 'GET';
  const response = await fetch(`${config.server.baseUrl}/${opts.path}`, opts);
  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`);
  }

  const reader = response.body?.getReader();

  if (!reader) {
    throw new Error('Failed to get reader from response body');
  }

  const decoder = new TextDecoder('utf-8');
  let done = false;
  while (!done) {
    const { done: doneReading, value } = await reader.read();
    done = doneReading;
    const result: string = decoder.decode(value);
    try {
      const res = JSON.parse(result) as ChatResponse;
      yield res;
    } catch (e: unknown) {
      if (e instanceof Error) {
        if (e instanceof SyntaxError && result.trim() === '') {
          break;
        }
        console.error("Error parsing JSON:", e);
      } else {
        console.error("Unknown error parsing JSON:", e);
      }
    }
  }
}

export async function req<T>(opts: RequestOptions): Promise<T> {
  const controller = new AbortController();
  opts.signal = controller.signal;

  // Cancel previous requests with the same key if specified
  if (opts.requestKey && pendingRequests[opts.requestKey]) {
    pendingRequests[opts.requestKey].abort();
  }

  if (opts.requestKey) {
    pendingRequests[opts.requestKey] = controller;
  }

  opts.headers = {
    ...opts.headers,
    'Content-Type': 'application/json'
  }
  opts.method = opts.method || 'GET';

  try {
    if (opts.timeout) {
      setTimeout(() => {
        if (opts.requestKey && pendingRequests[opts.requestKey]) {
          pendingRequests[opts.requestKey].abort();
          delete pendingRequests[opts.requestKey];
        }
      }, opts.timeout);
    }

    const response = await fetch(`${config.server.baseUrl}/${opts.path}`, opts);
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    if (opts.requestKey) {
      delete pendingRequests[opts.requestKey];
    }

    if (opts.method === 'DELETE') {
      return await response.text() as unknown as T;
    }

    return await response.json();
  } catch (error: unknown) {
    if (opts.requestKey) {
      delete pendingRequests[opts.requestKey];
    }

    if (error instanceof Error) {
      if (error.name === 'AbortError') {
        console.log('Request was cancelled');
      } else {
        console.error("Error in request:", error);
      }
    } else {
      console.error("Unknown error in request:", error);
    }

    throw error;
  }
}

export const getHeaders = (accessToken: string) => ({
  Authorization: `Bearer ${accessToken}`,
  'Content-Type': 'application/json'
});

export const debounce = <T extends (...args: never[]) => unknown>(func: T, waitFor: number) => {
  let timeout: ReturnType<typeof setTimeout> | null = null;

  const debounced = (...args: Parameters<T>) => {
    if (timeout !== null) {
      clearTimeout(timeout);
      timeout = null;
    }
    timeout = setTimeout(() => func(...args), waitFor);
  };

  return debounced as (...args: Parameters<T>) => void;
}


// Track pending requests
const pendingRequests: Record<string, AbortController> = {};