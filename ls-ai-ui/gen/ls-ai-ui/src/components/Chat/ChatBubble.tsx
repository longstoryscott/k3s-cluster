import React from 'react';
import { Box, Typography } from '@mui/material';

interface ChatBubbleProps {
  message: string;
  sender: 'user' | 'bot';
}

const ChatBubble: React.FC<ChatBubbleProps> = ({ message, sender }) => {
  const isUser = sender === 'user';

  return (
    <Box
      sx={{
        maxWidth: '70%',
        margin: isUser ? '10px auto 10px 0' : '10px 0 10px auto',
        padding: '10px',
        borderRadius: '10px',
        backgroundColor: isUser ? '#dcf8c6' : '#f1f0f0',
        alignSelf: isUser ? 'flex-end' : 'flex-start',
        display: 'flex',
        flexDirection: 'column'
      }}
    >
      <Typography variant="body1" sx={{ whiteSpace: 'pre-wrap' }}>
        {message}
      </Typography>
    </Box>
  );
};

export default ChatBubble;