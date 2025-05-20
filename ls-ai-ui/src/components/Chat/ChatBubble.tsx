import React, { memo, useState } from 'react';
import { Box, Paper, Typography, Link, Table, TableBody, TableCell, TableHead, TableRow, useTheme, Button } from '@mui/material';
import { ChatMessage, ChatUserMessage } from '../../api/types';
import ReactMarkdown from 'react-markdown';
import { Prism as SyntaxHighlighter, SyntaxHighlighterProps } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';
import remarkGfm from 'remark-gfm';
import remarkMath from 'remark-math';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import ReplayIcon from '@mui/icons-material/Replay';
import { useChat } from '../../chat';
import rehypeKatex from 'rehype-katex';
import 'katex/dist/katex.min.css';

// Utility function to replace Unicode characters that cause LaTeX compatibility issues
const sanitizeForLaTeX = (text: string): string => {
  if (!text) {
    return '';
  }
  
  // Map of problematic Unicode characters to their LaTeX-compatible replacements
  const replacements: Record<string, string> = {
    '\u2013': '-', // en dash (–) to hyphen
    '\u2014': '---', // em dash (—) to three hyphens
    '\u2018': "'", // left single quotation mark (') to apostrophe
    '\u2019': "'", // right single quotation mark (') to apostrophe
    '\u201C': '"', // left double quotation mark (") to straight double quote
    '\u201D': '"', // right double quotation mark (") to straight double quote
    '\u2026': '...' // horizontal ellipsis (…) to three dots
    // Add more replacements as needed
  };
  
  // Replace each problematic character
  return Object.entries(replacements).reduce(
    (result, [unicodeChar, replacement]) => 
      result.replace(new RegExp(unicodeChar, 'g'), replacement),
    text
  );
};

interface ChatBubbleProps {
  message: ChatMessage;
  inProgress?: boolean;
}

const extractThinkSection = (content: string) => {
  // Check for <think> tags
  const startIdx = content.indexOf('<think>');
  
  if (startIdx === -1) {
    // No <think> tag found
    return { think: null, rest: content };
  }
  
  // Look for closing tag
  const endIdx = content.indexOf('</think>', startIdx);
  
  // If we have both opening and closing tags
  if (endIdx !== -1) {
    const thinkContent = content.substring(startIdx + 7, endIdx).trim();
    // Combine text before <think> and after </think> as the main message
    const beforeThink = content.substring(0, startIdx).trim();
    const afterThink = content.substring(endIdx + 8).trim();
    const restContent = [beforeThink, afterThink].filter(Boolean).join('\n\n');
    
    return {
      think: thinkContent,
      rest: restContent || '' // Ensure we don't return null for rest
    };
  }  else {
    // Everything before <think> is regular content
    const beforeThink = content.substring(0, startIdx).trim();
    // Everything after <think> goes into the think section
    const thinkContent = content.substring(startIdx + 7).trim();
    
    return {
      think: thinkContent,
      rest: beforeThink || '' // Ensure we don't return null for rest
    };
  }
};

const ChatBubble: React.FC<ChatBubbleProps> = memo(({ message, inProgress = false }) => {
  const isUser = message.role === 'user';
  const isError = message.status === 'error';
  const theme = useTheme();
  const { retryMessage } = useChat();
  const [showThink, setShowThink] = useState(false);
  const { think, rest } = extractThinkSection(message.content);

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
          borderLeft: `${theme.spacing(0.5)} solid`,
          borderLeftColor: isUser ? theme.palette.secondary.main : isError ? theme.palette.error.main : theme.palette.primary.main,
          textAlign: 'left'
        }}
      >
        {think && (
          <Box sx={{ mb: 1 }}>
            <Button
              size="small"
              variant="outlined"
              onClick={() => setShowThink((v) => !v)}
              sx={{ textTransform: 'none', fontSize: '0.8em', mb: 0.5 }}
            >
              {showThink ? 'Hide' : 'Show'} internal notes
            </Button>
            {showThink && (
              <Box sx={{
                bgcolor: theme.palette.background.paper,
                border: `1px solid ${theme.palette.grey[300]}`,
                borderRadius: theme.shape.borderRadius,
                p: 1,
                mt: 0.5,
                fontSize: '0.9em',
                color: theme.palette.text.secondary,
                whiteSpace: 'pre-wrap'
              }}>
                <ReactMarkdown
                  remarkPlugins={[remarkGfm, remarkMath]}
                  rehypePlugins={[rehypeKatex]}
                >
                  {sanitizeForLaTeX(think)}
                </ReactMarkdown>
              </Box>
            )}
          </Box>
        )}

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
          remarkPlugins={[remarkGfm, remarkMath]}
          rehypePlugins={[rehypeKatex]}
          components={{
            // Code blocks with syntax highlighting
            code({node, className, children, ...props}) {
              const match = /language-(\w+)/.exec(className || '');
              return (className ? (
                <SyntaxHighlighter
                  style={vscDarkPlus}
                  language={match?.[1] || 'text'}
                  wrapLines={true}
                  showLineNumbers={!!className}
                  {...props as SyntaxHighlighterProps}
                >
                  {String(children).replace(/\n$/, '')}
                </SyntaxHighlighter>) : 
                (
                  <Typography
                    component="span"
                    sx={{
                      fontFamily:'monospace',
                      backgroundColor: theme.palette.background.paper,
                      px: theme.spacing(0.5),
                      py: theme.spacing(0.25),
                      fontSize: '0.9em',
                      color: theme.palette.text.primary
                    }}
                    {...props}
                  >{String(children).replace(/\n$/, '')}</Typography>
                )
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
          {sanitizeForLaTeX(rest)}
        </ReactMarkdown>
      </Paper>
    </Box>
  );
});

export default ChatBubble;