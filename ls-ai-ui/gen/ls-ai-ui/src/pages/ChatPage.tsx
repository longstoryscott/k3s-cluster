import React, { useEffect, useState } from 'react';
import { useAuth } from '../auth/AuthProvider';
import { ChatContainer } from '../components/Chat/ChatContainer';
import { ChatHeader } from '../components/Chat/ChatHeader';
import { ChatInput } from '../components/Chat/ChatInput';
import { useChat } from '../hooks/useChat';
import { MainLayout } from '../components/Layout/MainLayout';

const ChatPage = () => {
  const auth = useAuth();
  const { messages, sendMessage, fetchMessages } = useChat();
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const loadMessages = async () => {
      await fetchMessages();
      setLoading(false);
    };
    loadMessages();
  }, [fetchMessages]);

  const handleSendMessage = async (message) => {
    await sendMessage(message);
  };

  return (
    <MainLayout>
      <ChatHeader />
      {loading ? (
        <div>Loading...</div>
      ) : (
        <ChatContainer messages={messages} />
      )}
      <ChatInput onSend={handleSendMessage} />
    </MainLayout>
  );
};

export default ChatPage;