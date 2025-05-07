import { useEffect } from 'react';
import { List, Typography, Box, useTheme } from '@mui/material';
import ChatItem from './ChatItem';
import { useChat } from '../../chat';

const ChatHistory = () => {
  const { conversations, fetchConversations, selectConversation } = useChat();
  const theme = useTheme();

  useEffect(() => {
    // Load conversations when component mounts
    fetchConversations();
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleSelectChat = (chatId: number) => {
    selectConversation(chatId);
  };

  return (
    <Box>
      <Typography variant="subtitle1" sx={{ mb: theme.spacing(1) }}>
        Recent Conversations
      </Typography>
      
      {conversations?.length ? (
        <List sx={{ overflow: 'auto', maxHeight: 'calc(100vh - 300px)' }}>
          {conversations.map((chat) => (
            <ChatItem
              key={chat.id}
              chatId={chat.id!}
              chatTitle={chat.title || `Chat ${chat.id}`}
              onSelect={handleSelectChat}
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