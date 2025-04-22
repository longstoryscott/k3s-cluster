import { useState, useEffect } from 'react';
import { gen } from '../api';
import { useAuth } from '../auth/AuthProvider';
import { ChatMessage } from '../../../../src/api/types';

export const useChat = () => {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const auth = useAuth();

  const sendMessage = async (prompt: string) => {
    setLoading(true);
    setError(null);

    try {
      const generator = gen({
        body: JSON.stringify({
          model: 'gemma3:1b',
          messages: [{ role: 'user', content: prompt }]
        }),
        method: 'POST',
        headers: {
          Authorization: `Bearer ${auth.user?.accessToken}`,
          'Content-Type': 'application/json'
        },
        path: 'api/chat'
      });

      for await (const res of generator) {
        if (res.message) {
          setMessages((prev) => {
            const state = prev;
            state.push(res.message);

            return state;
          });
        }
        if (res.done) {
          break;
        }
      }
    } catch (err: unknown) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  };

  return {
    messages,
    loading,
    error,
    sendMessage
  };
};