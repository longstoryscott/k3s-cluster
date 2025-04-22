import React from 'react';
import { CircularProgress, Typography } from '@mui/material';

const TypingIndicator = () => {
  return (
    <div style={{ display: 'flex', alignItems: 'center', padding: '10px' }}>
      <CircularProgress size={24} />
      <Typography variant="body2" style={{ marginLeft: '8px' }}>
        Typing...
      </Typography>
    </div>
  );
};

export default TypingIndicator;