import { List, Typography, Box, useTheme } from '@mui/material';
import ChatItem from './ChatItem';
import { useChat } from '../../chat';

const ChatHistory = () => {
  const { conversations } = useChat();
  const theme = useTheme();
  return (
    <Box>
      <Typography variant="subtitle1" sx={{ mb: theme.spacing(1) }}>
        Recent Conversations
      </Typography>
      
      {conversations?.length ? (
        <List sx={{ overflow: 'auto' }}>
          {conversations.map((chat) => (
            <ChatItem
              key={chat.id}
              chatId={chat.id!}
              chatTitle={chat.title || `Chat ${chat.id}`}
            />
          ))}
        </List>
      ) : (
        <Typography 
          variant="body2" 
          color="text.secondary"
          sx={{ textAlign: 'center', mt: theme.spacing(2) }}
        >
          No conversation history
        </Typography>
      )}
    </Box>
  );
};

export default ChatHistory;