// filepath: /Users/scott.long/workspace/k3s-cluster/ls-ai-ui/src/api/index.ts
import config from "../config";
import { BodyDeserialized, ChatResponse } from "../../../../src/api/types";

export type RequestOptions = {
  method?: 'POST' | 'GET' | 'PUT' | 'DELETE';
  headers?: HeadersInit;
  body?: string;
  path: string;
};

export async function* gen(opts: RequestOptions): AsyncGenerator<ChatResponse> {
  opts.headers = {
    ...opts.headers,
    'Content-Type': 'application/json'
  };
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
    } catch (e) {
      console.error(e);
    }
  }
}