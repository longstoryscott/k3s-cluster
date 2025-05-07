import React, { useState } from 'react';
import { TextField, Button, Box, Typography, useTheme } from '@mui/material';
import { useChat } from '../../chat';

const ChatInput = () => {
  const [input, setInput] = useState('');
  const { sendMessage, isTyping, currentConversation } = useChat();
  const theme = useTheme();
  
  // Check if there's an active conversation
  const hasConversation = !!currentConversation?.id;

  const handleKeyPress = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'Enter' && !event.shiftKey) {
      event.preventDefault();
      handleSend();
    }
  };

  const handleSend = () => {
    const trimmedInput = input.trim();
    if (trimmedInput && !isTyping && hasConversation) {
      sendMessage({ content: trimmedInput, role: 'user', conversationId: currentConversation.id! });
      setInput('');
    }
  };

  return (
    <Box 
      sx={{ 
        display: 'flex', 
        flexDirection: 'column',
        p: theme.spacing(2),
        borderTop: `${theme.spacing(0.125)} solid`,
        borderColor: theme.palette.divider
      }}
    >
      {!hasConversation && (
        <Typography 
          variant="body2" 
          color="text.secondary" 
          sx={{ mb: theme.spacing(1), textAlign: 'center' }}
        >
          Start a new conversation to begin chatting
        </Typography>
      )}
      <Box sx={{ display: 'flex', alignItems: 'center' }}>
        <TextField
          variant="outlined"
          fullWidth
          placeholder={hasConversation ? "Type your message..." : "No active conversation..."}
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={handleKeyPress}
          multiline
          maxRows={4}
          disabled={isTyping || !hasConversation}
        />
        <Button 
          onClick={handleSend} 
          variant="contained" 
          color="primary" 
          sx={{ ml: theme.spacing(2) }}
          disabled={!input.trim() || isTyping || !hasConversation}
          type='submit'
        >
          Send
        </Button>
      </Box>
    </Box>
  );
};

export default ChatInput;