import React, { useState } from 'react';
import { TextField, Button, Box } from '@mui/material';
import { useAuth } from '../../auth/AuthProvider';
import { gen } from '../../api';

const ChatInput = ({ onSend }) => {
  const [input, setInput] = useState('');
  const auth = useAuth();

  const handleSend = async () => {
    if (input.trim()) {
      const responseGenerator = gen({
        body: JSON.stringify({
          model: 'gemma3:1b',
          messages: [{ role: 'user', content: input }]
        }),
        method: 'POST',
        headers: {
          Authorization: `Bearer ${auth.user?.accessToken}`,
          'Content-Type': 'application/json'
        },
        path: 'api/chat'
      });

      for await (const res of responseGenerator) {
        onSend(res);
        if (res.done) {
          break;
        }
      }

      setInput('');
    }
  };

  const handleKeyPress = (event) => {
    if (event.key === 'Enter') {
      handleSend();
    }
  };

  return (
    <Box display="flex" alignItems="center" mt={2}>
      <TextField
        variant="outlined"
        fullWidth
        placeholder="Type your message..."
        value={input}
        onChange={(e) => setInput(e.target.value)}
        onKeyPress={handleKeyPress}
      />
      <Button onClick={handleSend} variant="contained" color="primary" sx={{ ml: 2 }}>
        Send
      </Button>
    </Box>
  );
};

export default ChatInput;