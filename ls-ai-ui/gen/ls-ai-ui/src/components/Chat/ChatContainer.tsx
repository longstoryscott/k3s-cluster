import React, { useEffect, useState } from 'react';
import { useAuth } from '../../auth/AuthProvider';
import { useChat } from '../../hooks/useChat';
import ChatHeader from './ChatHeader';
import ChatInput from './ChatInput';
import ChatBubble from './ChatBubble';
import TypingIndicator from './TypingIndicator';
import { Box } from '@mui/material';

const ChatContainer = () => {
  const { user } = useAuth();
  const { messages, sendMessage, isTyping } = useChat();
  const [inputValue, setInputValue] = useState('');

  const handleSendMessage = async () => {
    if (inputValue.trim()) {
      await sendMessage(inputValue);
      setInputValue('');
    }
  };

  useEffect(() => {
    // Scroll to the bottom of the chat when messages change
    const chatContainer = document.getElementById('chat-container');
    if (chatContainer) {
      chatContainer.scrollTop = chatContainer.scrollHeight;
    }
  }, [messages]);

  return (
    <Box id="chat-container" sx={{ height: '100%', overflowY: 'auto', padding: 2 }}>
      <ChatHeader />
      {messages.map((msg, index) => (
        <ChatBubble key={index} message={msg} isUser={msg.sender === user?.email} />
      ))}
      {isTyping && <TypingIndicator />}
      <ChatInput value={inputValue} onChange={(e) => setInputValue(e.target.value)} onSend={handleSendMessage} />
    </Box>
  );
};

export default ChatContainer;