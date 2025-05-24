import React, { useState } from 'react';
import { Box, Button, useTheme, Typography, Paper } from '@mui/material';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import remarkMath from 'remark-math';
import rehypeKatex from 'rehype-katex';
import 'katex/dist/katex.min.css';
import { sanitizeForLaTeX } from './utils';
import SearchIcon from '@mui/icons-material/Search';
import LoadingAnimation from '../Shared/LoadingAnimation';

interface ThinkSectionProps {
  think?: string;
  thinking?: boolean;
  searching?: boolean;
  text?: string;
}

const ThinkSection: React.FC<ThinkSectionProps> = ({ think, thinking, searching, text }) => {
  const [showThink, setShowThink] = useState(false);
  const theme = useTheme();

  if (!thinking && !searching && !think) {
    return null;
  }

  return (
    <Paper
      elevation={1}
      sx={{
        p: 2,
        mb: 2,
        display: 'flex',
        alignItems: 'center',
        backgroundColor: 'rgba(0, 0, 0, 0.03)',
        maxWidth: '80%',
        alignSelf: 'flex-start',
        borderRadius: 2
      }}
    >
      {searching ? (
        <SearchIcon sx={{ mr: 1, color: 'text.secondary' }} />
      ) : (
        <LoadingAnimation size={20} sx={{ mr: 1 }} />
      )}
      
      <Typography variant="body2" color="text.secondary">
        {text || (searching ? 'Searching the web...' : 'Thinking...')}
      </Typography>

      {think && (
        <Box sx={{ mb: 1, ml: 2 }}>
          <Button
            size="small"
            variant="outlined"
            onClick={() => setShowThink((v) => !v)}
            sx={{ textTransform: 'none', fontSize: '0.8em', mb: 0.5 }}
          >
            {showThink ? 'Hide' : 'Show'} thoughts
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
    </Paper>
  );
};

export default ThinkSection;
