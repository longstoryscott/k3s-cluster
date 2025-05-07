import { Box, Divider, Typography, useTheme } from '@mui/material';
import { useAuth } from '../../auth';
import ChatHistory from '../Sidebar/ChatHistory';
import NewChatButton from '../Sidebar/NewChatButton';
import Navigation from './Navigation';

const Sidebar = () => {
  const { user } = useAuth();
  const theme = useTheme();

  return (
    <Box
      sx={{
        width: 250,
        bgcolor: 'background.paper',
        height: '100vh',
        display: 'flex',
        flexDirection: 'column',
        padding: theme.spacing(2)
      }}
    >
      <Typography variant="h6" sx={{ mb: theme.spacing(2) }}>
        Welcome, {user.name || 'User'}
      </Typography>
      
      <Navigation />
      
      <Box sx={{ mt: theme.spacing(2), mb: theme.spacing(2) }}>
        <NewChatButton />
      </Box>
      
      <Divider sx={{ my: theme.spacing(1) }} />
      
      <Box sx={{ flex: 1, overflow: 'auto' }}>
        <ChatHistory />
      </Box>
    </Box>
  );
};

export default Sidebar;