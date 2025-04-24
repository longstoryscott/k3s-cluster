import React from 'react';
import { Box, Paper, Typography } from '@mui/material';
import { ChatMessage } from '../../api/types';
import ReactMarkdown from 'react-markdown';
import { Prism as SyntaxHighlighter, SyntaxHighlighterProps } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';

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
          maxWidth: '70%',
          backgroundColor: isUser ? 'primary.light' : 'background.paper',
          color: isUser ? 'primary.contrastText' : 'text.primary',
          borderRadius: 2,
          opacity: inProgress ? 0.9 : 1
        }}
      >
        <Typography 
          variant="subtitle2" 
          sx={{ fontWeight: 'bold', mb: 1 }}
        >
          {isUser ? 'You' : 'Assistant'}
          {inProgress ? ' (typing...)' : ''}
        </Typography>
        
        <ReactMarkdown
          components={{
            code({node, className, children, ...props}) {
              const match = /language-(\w+)/.exec(className || '');
              return !className ? (
                <code {...props}>{children}</code>
              ) : (
                <SyntaxHighlighter
                  style={vscDarkPlus}
                  language={match?.[1] || 'text'}
                  PreTag="div"
                  {...props as SyntaxHighlighterProps}
                >
                  {String(children).replace(/\n$/, '')}
                </SyntaxHighlighter>
              );
            }
          }}
        >
          {message.content}
        </ReactMarkdown>
      </Paper>
    </Box>
  );
};

export default ChatBubble;