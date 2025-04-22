import React from 'react';
import { AppBar, Toolbar, Typography, IconButton } from '@mui/material';
import { Menu as MenuIcon } from '@mui/icons-material';

interface ChatHeaderProps {
  title: string;
  onMenuClick: React.MouseEventHandler<HTMLButtonElement>
}

const ChatHeader = ({ title, onMenuClick }: ChatHeaderProps) => (
  <AppBar position="static">
    <Toolbar>
      <IconButton edge="start" color="inherit" onClick={onMenuClick}>
        <MenuIcon />
      </IconButton>
      <Typography variant="h6" style={{ flexGrow: 1 }}>
        {title}
      </Typography>
    </Toolbar>
  </AppBar>
);


export default ChatHeader;