import React from 'react';
import { useChat } from '../../hooks/useChat';
import ChatItem from './ChatItem';
import { Box, Typography } from '@mui/material';

const ChatHistory = () => {
  const { chatHistory } = useChat();

  return (
    <Box sx={{ padding: 2 }}>
      <Typography variant="h6" gutterBottom>
        Chat History
      </Typography>
      {chatHistory.length === 0 ? (
        <Typography variant="body2" color="textSecondary">
          No chat history available.
        </Typography>
      ) : (
        chatHistory.map((chat, index) => (
          <ChatItem key={index} chat={chat} />
        ))
      )}
    </Box>
  );
};

export default ChatHistory;