import { Box, List, ListItem, ListItemText, Divider } from '@mui/material';
import { useAuth } from '../../auth';
import ChatHistory from '../Sidebar/ChatHistory';
import NewChatButton from '../Sidebar/NewChatButton';

const Sidebar = () => {
  const { user } = useAuth();

  return (
    <Box
      sx={{
        width: 250,
        bgcolor: 'background.paper',
        height: '100vh',
        display: 'flex',
        flexDirection: 'column',
        padding: 2
      }}
    >
      <h2>Welcome, {user.name || 'User'}</h2>
      <Divider />
      <List>
        <ListItem>
          <ListItemText primary="Chat History" />
        </ListItem>
        <ChatHistory />
        <NewChatButton />
      </List>
    </Box>
  );
};

export default Sidebar;