import React from 'react';
import { Button } from '@mui/material';
import { useChat } from '../../hooks/useChat';

const NewChatButton = () => {
  const { startNewChat } = useChat();

  const handleNewChat = () => {
    startNewChat();
  };

  return (
    <Button 
      variant="contained" 
      color="primary" 
      onClick={handleNewChat} 
      fullWidth
    >
      New Chat
    </Button>
  );
};

export default NewChatButton;