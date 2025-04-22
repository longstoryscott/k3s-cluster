import React from 'react';
import { Box } from '@mui/material';
import Sidebar from './Sidebar';
import TopBar from './TopBar';

const MainLayout: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  return (
    <Box display="flex" height="100vh">
      <Sidebar />
      <Box flexGrow={1}>
        <TopBar />
        <Box p={2} height="calc(100% - 64px)" overflow="auto">
          {children}
        </Box>
      </Box>
    </Box>
  );
};

export default MainLayout;