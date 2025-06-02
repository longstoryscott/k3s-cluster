import React, { memo } from 'react';
import { Box, Paper, Button, Fade } from '@mui/material';
import { ChatMessage, ChatUserMessage } from '../../api/types';
import { useChat } from '../../chat';
import ReplayIcon from '@mui/icons-material/Replay';
import MarkdownRenderer from '../Shared/MarkdownRenderer';
import ThinkSection from './ThinkSection';
import ErrorMessage from './ErrorMessage';
import { sanitizeForLaTeX, parseResponse } from './utils';

interface ChatBubbleProps {
  message: ChatMessage;
}

const ChatBubble: React.FC<ChatBubbleProps> = memo(({ message }) => {
  const { retryMessage, isLoading, isTyping } = useChat();
  const inProgress = isLoading || isTyping;
  const { think, rest } = parseResponse(message.content, isTyping);
  const isUser = message.role === 'user';
  const isError = message.status === 'error';
  
  const handleRetry = () => {
    if (isError && isUser && 'conversationId' in message) {
      retryMessage(message as ChatUserMessage);
    }
  };

  return (
    <Box
      sx={{
        display: 'flex',
        justifyContent: isUser ? 'flex-end' : 'flex-start',
        mb: 2
      }}
    >
      <Fade in={true} timeout={1500}>
        <Paper
          sx={{
            p: { xs: 1.5, sm: 2 },
            width: { xs: '100%', sm: isUser ? '80%' : '90%' },
            backgroundColor: isUser ? 'primary.light' : 'background.paper',
            color: isUser ? 'primary.contrastText' : 'text.primary',
            borderRadius: 2,
            opacity: inProgress ? 0.75 : 1,
            borderLeft: `0.5px solid`,
            borderLeftColor: isUser ? 'secondary.main' : isError ? 'error.main' : 'primary.main',
            wordBreak: 'break-word',
            overflowWrap: 'break-word',
            minHeight: 100
          }}
        >
          {!isUser && (think || inProgress) && <ThinkSection think={think || ""} inProgress={inProgress} />}
          <Box sx={{ 
            display: 'flex', 
            justifyContent: 'space-between', 
            alignItems: 'center', 
            mb: 0.5,
            flexWrap: 'wrap'
          }}>
            {isError && isUser && (
              <Button
                startIcon={<ReplayIcon />}
                color="error"
                size="small"
                onClick={handleRetry}
                variant="outlined"
                sx={{ 
                  ml: 1,
                  minHeight: 36,
                  mt: { xs: 0.5, sm: 0 }
                }}
              >
                Retry
              </Button>
            )}
          </Box>

          {isError && isUser && (
            <ErrorMessage message="Failed to send. Click retry to try again." />
          )}

          <MarkdownRenderer sanitizeForLaTeX={sanitizeForLaTeX}>
            {rest}
          </MarkdownRenderer>
        </Paper>
      </Fade>
    </Box>
  );
});

export default ChatBubble;