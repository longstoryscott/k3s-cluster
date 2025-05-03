import React from 'react';
import { Box, Paper, Typography, Link, Table, TableBody, TableCell, TableHead, TableRow } from '@mui/material';
import { ChatMessage } from '../../api/types';
import ReactMarkdown from 'react-markdown';
import { Prism as SyntaxHighlighter, SyntaxHighlighterProps } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';
import remarkGfm from 'remark-gfm';

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
        justifyContent: isUser ? 'flex-end' : 'flex-start', // Restore original positioning
        mb: 2
      }}
    >
      <Paper
        elevation={1}
        sx={{
          p: 2,
          maxWidth: '75%',
          backgroundColor: isUser ? 'primary.light' : 'background.paper',
          color: isUser ? 'primary.contrastText' : 'text.primary',
          borderRadius: 2,
          opacity: inProgress ? 0.9 : 1,
          borderLeft: isUser ? 'none' : '4px solid',
          borderLeftColor: isUser ? undefined : 'primary.main',
          textAlign: 'left' // Ensure text inside bubbles is left-aligned
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
          remarkPlugins={[remarkGfm]} // Adds support for tables, strikethrough, etc.
          components={{
            // Code blocks with syntax highlighting
            code({node, className, children, ...props}) {
              const match = /language-(\w+)/.exec(className || '');
              return !className ? (
                <code style={{ backgroundColor: '#f0f0f0', padding: '0.2em 0.4em', borderRadius: '3px' }} {...props}>
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
                <Box sx={{ overflowX: 'auto', my: 2 }}>
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
                    borderLeft: '4px solid',
                    borderColor: 'grey.400',
                    pl: 2,
                    py: 0.5,
                    my: 1,
                    bgcolor: 'grey.100',
                    borderRadius: 1
                  }}
                  {...props}
                >
                  {children}
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