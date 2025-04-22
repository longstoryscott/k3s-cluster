import React from 'react';
import { Box } from '@mui/material';

interface CodeBlockProps {
  code: string;
  language?: string;
}

const CodeBlock: React.FC<CodeBlockProps> = ({ code }) => {
  return (
    <Box
      component="pre"
      sx={{
        backgroundColor: '#f5f5f5',
        borderRadius: '4px',
        padding: '16px',
        overflowX: 'auto',
        whiteSpace: 'pre-wrap',
        wordWrap: 'break-word',
        fontFamily: 'monospace',
        fontSize: '14px'
      }}
    >
      <code>{code}</code>
    </Box>
  );
};

export default CodeBlock;