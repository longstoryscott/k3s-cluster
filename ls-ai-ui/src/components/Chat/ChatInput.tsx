import React, { useState } from 'react';
import { TextField, Button, Box } from '@mui/material';
import { useChat } from '../../chat';

const ChatInput = () => {
  const [input, setInput] = useState('');
  const { sendMessage, isTyping } = useChat();

  const handleKeyPress = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'Enter' && !event.shiftKey) {
      event.preventDefault();
      handleSend();
    }
  };

  const handleSend = () => {
    const trimmedInput = input.trim();
    if (trimmedInput && !isTyping) {
      sendMessage({ content: trimmedInput, role: 'user' });
      setInput('');
    }
  };

  return (
    <Box 
      sx={{ 
        display: 'flex', 
        alignItems: 'center', 
        p: 2,
        borderTop: '1px solid',
        borderColor: 'divider'
      }}
    >
      <TextField
        variant="outlined"
        fullWidth
        placeholder="Type your message..."
        value={input}
        onChange={(e) => setInput(e.target.value)}
        onKeyDown={handleKeyPress}
        multiline
        maxRows={4}
        disabled={isTyping}
      />
      <Button 
        onClick={handleSend} 
        variant="contained" 
        color="primary" 
        sx={{ ml: 2 }}
        disabled={!input.trim() || isTyping}
        type='submit'
      >
        Send
      </Button>
    </Box>
  );
};

export default ChatInput;