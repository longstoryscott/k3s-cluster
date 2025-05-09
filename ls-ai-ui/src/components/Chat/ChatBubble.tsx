import React from 'react';
import { Box, Paper, Typography, Link, Table, TableBody, TableCell, TableHead, TableRow, useTheme, Button } from '@mui/material';
import { ChatMessage, ChatUserMessage } from '../../api/types';
import ReactMarkdown from 'react-markdown';
import { Prism as SyntaxHighlighter, SyntaxHighlighterProps } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';
import remarkGfm from 'remark-gfm';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import ReplayIcon from '@mui/icons-material/Replay';
import { useChat } from '../../chat';

interface ChatBubbleProps {
  message: ChatMessage;
  inProgress?: boolean;
}

const ChatBubble: React.FC<ChatBubbleProps> = ({ message, inProgress = false }) => {
  const isUser = message.role === 'user';
  const isError = message.status === 'error';
  const theme = useTheme();
  const { retryMessage } = useChat();

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
      <Paper
        elevation={1}
        sx={{
          p: theme.spacing(2),
          maxWidth: '75%',
          backgroundColor: isUser ? theme.palette.primary.light : theme.palette.background.paper,
          color: isUser ? theme.palette.primary.contrastText : theme.palette.text.primary,
          borderRadius: theme.shape.borderRadius * 2,
          opacity: inProgress ? 0.9 : 1,
          borderLeft: isUser ? 'none' : `${theme.spacing(0.5)} solid`,
          borderLeftColor: isUser ? undefined : isError ? theme.palette.error.main : theme.palette.primary.main,
          textAlign: 'left'
        }}
      >
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: theme.spacing(1) }}>
          <Typography 
            variant="subtitle2" 
            sx={{ fontWeight: 'bold' }}
          >
            {isUser ? 'You' : 'Assistant'}
            {inProgress ? ' (typing...)' : ''}
          </Typography>
          
          {isError && isUser && (
            <Button
              startIcon={<ReplayIcon />}
              color="error"
              size="small"
              onClick={handleRetry}
              variant="outlined"
              sx={{ ml: theme.spacing(1) }}
            >
              Retry
            </Button>
          )}
        </Box>

        {isError && isUser && (
          <Box sx={{ 
            display: 'flex', 
            alignItems: 'center', 
            color: theme.palette.error.main, 
            mb: theme.spacing(1),
            p: theme.spacing(1),
            bgcolor: theme.palette.error.light,
            borderRadius: theme.shape.borderRadius
          }}>
            <ErrorOutlineIcon fontSize="small" sx={{ mr: theme.spacing(1) }} />
            <Typography variant="caption">Failed to send. Click retry to try again.</Typography>
          </Box>
        )}
        
        <ReactMarkdown
          remarkPlugins={[remarkGfm]}
          components={{
            // Code blocks with syntax highlighting
            code({node, className, children, ...props}) {
              const match = /language-(\w+)/.exec(className || '');
              return !className ? (
                <code style={{ 
                  backgroundColor: theme.palette.background.paper, 
                  padding: `${theme.spacing(0.25)} ${theme.spacing(0.5)}`, 
                  borderRadius: theme.shape.borderRadius / 2
                }} {...props}>
                  {children}
                </code>
              ) : (
                <SyntaxHighlighter
                  style={vscDarkPlus}
                  language={match?.[1] || 'text'}
                  PreTag="div"
                  wrapLines={true}
                  showLineNumbers={true}
                  {...props as SyntaxHighlighterProps}
                >
                  {String(children).replace(/\n$/, '')}
                </SyntaxHighlighter>
              );
            },
            // Enhanced link component
            a({node, children, href, ...props}) {
              return (
                <Link 
                  href={href} 
                  target="_blank" 
                  rel="noopener noreferrer"
                  color="primary"
                  underline="hover"
                  {...props}
                >
                  {children}
                </Link>
              );
            },
            // Table components
            table({node, children, ...props}) {
              return (
                <Box sx={{ overflowX: 'auto', my: theme.spacing(2) }}>
                  <Table size="small" {...props}>
                    {children}
                  </Table>
                </Box>
              );
            },
            thead({node, children, ...props}) {
              return <TableHead {...props}>{children}</TableHead>;
            },
            tbody({node, children, ...props}) {
              return <TableBody {...props}>{children}</TableBody>;
            },
            tr({node, children, ...props}) {
              return <TableRow {...props}>{children}</TableRow>;
            },
            th({node, children, ...props}) {
              // @ts-expect-error ts(2322)
              return <TableCell component="th" sx={{ fontWeight: 'bold' }} {...props}>{children}</TableCell>;
            },
            td({node, children, ...props}) {
              // @ts-expect-error ts(2322)
              return <TableCell {...props}>{children}</TableCell>;
            },
            // Blockquotes
            blockquote({node, children, ...props}) {
              return (
                <Box 
                  component="blockquote"
                  sx={{ 
                    borderLeft: `${theme.spacing(0.5)} solid`,
                    borderColor: theme.palette.grey[400],
                    pl: theme.spacing(2),
                    py: theme.spacing(0.5),
                    my: theme.spacing(1),
                    bgcolor: theme.palette.grey[100],
                    borderRadius: theme.shape.borderRadius / 4
                  }}
                  {...props}
                >
                  {children}
                </Box>
              );
            },

            img({node, alt, src, ...props}) {
              return (
                <Box sx={{ textAlign: 'center', my: theme.spacing(2) }}>
                  <img 
                    src={src} 
                    alt={alt} 
                    style={{ 
                      maxWidth: '100%', 
                      borderRadius: theme.shape.borderRadius 
                    }} 
                    {...props} 
                  />
                </Box>
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