import React from 'react';
import { Paper, Box, Typography } from '@mui/material';
import ReactMarkdown from 'react-markdown';
import { ChatMessage } from '../../api/types';

interface ChatBubbleProps {
  message: ChatMessage;
  inProgress?: boolean;
}

const ChatBubble: React.FC<ChatBubbleProps> = ({ message, inProgress = false }) => {
  const isUser = message.role === 'user';
  
  return (
    <Box
      sx={{
        display: 'flex',
        justifyContent: isUser ? 'flex-end' : 'flex-start',
        mb: 2
      }}
    >
      <Paper
        elevation={1}
        sx={{
          p: 2,
          maxWidth: '80%',
          backgroundColor: isUser ? 'primary.light' : 'background.paper',
          borderRadius: 2,
          ...(inProgress && { borderLeft: '2px solid orange' })
        }}
      >
        <Typography variant="subtitle2" gutterBottom>
          {isUser ? 'You:' : 'Assistant:'}
        </Typography>
        
        <Box className="markdown-content">
          <ReactMarkdown>
            {message.content}
          </ReactMarkdown>
        </Box>
      </Paper>
    </Box>
  );
};

export default ChatBubble;