import React, { memo } from 'react';
import { Box, Paper, Button, Fade, useTheme } from '@mui/material';
import { ChatMessage, ChatUserMessage } from '../../api/types';
import { useChat } from '../../chat';
import ReplayIcon from '@mui/icons-material/Replay';
import MarkdownRenderer from '../Shared/MarkdownRenderer';
import 'katex/dist/katex.min.css';
import ThinkSection from './ThinkSection';
import ErrorMessage from './ErrorMessage';
import { sanitizeForLaTeX, parseResponse } from './utils';

interface ChatBubbleProps {
  message: ChatMessage;
  inProgress?: boolean;
}

const ChatBubble: React.FC<ChatBubbleProps> = memo(({ message, inProgress = false }) => {
  const isUser = message.role === 'user';
  const isError = message.status === 'error';
  const theme = useTheme();
  const { retryMessage } = useChat();
  const { think, rest } = parseResponse(message.content);

  const handleRetry = () => {
    // Only retry if this is an error message from the user that has a conversation ID
    if (isError && isUser && 'conversationId' in message) {
      retryMessage(message as ChatUserMessage);
    }
  };

  return (
    <Box
      sx={{
        display: 'flex',
        justifyContent: isUser ? 'flex-end' : 'flex-start',
        mb: theme.spacing(2)
      }}
    >
      <Fade in={true} timeout={1500}>
        <Paper
          elevation={1}
          sx={{
            p: { xs: theme.spacing(1.5), sm: theme.spacing(2) }, // Responsive padding
            width: { xs:  '100%', sm: isUser ? '80%' : '90%'  }, // Wider bubbles on mobile
            backgroundColor: isUser ? theme.palette.primary.light : theme.palette.background.paper,
            color: isUser ? theme.palette.primary.contrastText : theme.palette.text.primary,
            borderRadius: theme.shape.borderRadius * 2,
            opacity: inProgress ? 0.85 : 1,
            borderLeft: `${theme.spacing(0.5)} solid`,
            borderLeftColor: isUser ? theme.palette.secondary.main : isError ? theme.palette.error.main : theme.palette.primary.main,
            textAlign: 'left',
            wordBreak: 'break-word', // Prevent text overflow on small screens
            overflowWrap: 'break-word' // Handle long words
          }}
        >
          <ThinkSection think={think || ""} />
          <Box sx={{ 
            display: 'flex', 
            justifyContent: 'space-between', 
            alignItems: 'center', 
            mb: { xs: theme.spacing(0.5), sm: theme.spacing(1) },
            flexWrap: 'wrap' // Allow wrapping on very small screens
          }}>
            {isError && isUser && (
              <Button
                startIcon={<ReplayIcon />}
                color="error"
                size="small"
                onClick={handleRetry}
                variant="outlined"
                sx={{ 
                  ml: theme.spacing(1),
                  minHeight: '36px', // Touch-friendly height
                  mt: { xs: theme.spacing(0.5), sm: 0 } // Add spacing if wrapped
                }}
              >
                Retry
              </Button>
            )}
          </Box>

          {isError && isUser && (
            <ErrorMessage message="Failed to send. Click retry to try again." />
          )}

          {/* Use MarkdownRenderer for markdown rendering */}
          <MarkdownRenderer sanitizeForLaTeX={sanitizeForLaTeX}>
            {rest}
          </MarkdownRenderer>
        </Paper>
      </Fade>
    </Box>
  );
});

export default ChatBubble;