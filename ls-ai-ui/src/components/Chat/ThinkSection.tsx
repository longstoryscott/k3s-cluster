import React, { useState } from 'react';
import { Box, Button, useTheme } from '@mui/material';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import remarkMath from 'remark-math';
import rehypeKatex from 'rehype-katex';
import 'katex/dist/katex.min.css';
import { sanitizeForLaTeX } from './utils';

interface ThinkSectionProps {
  think: string;
}

const ThinkSection: React.FC<ThinkSectionProps> = ({ think }) => {
  const [showThink, setShowThink] = useState(false);
  const theme = useTheme();

  if (!think) {
    return null;
  }

  return (
    <Box sx={{ mb: 1 }}>
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
  );
};

export default ThinkSection;
