import React from 'react';
import { Box } from '@mui/material';
import Sidebar from './Sidebar';
import TopBar from './TopBar';

const MainLayout: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  return (
    <Box display="flex" height="100%">
      <Sidebar />
      <Box flexGrow={1}>
        <TopBar />
        <Box p={2} overflow="auto">
          {children}
        </Box>
      </Box>
    </Box>
  );
};

export default MainLayout;