import ChatItem from './ChatItem';
import { Box, Typography } from '@mui/material';
import { useChat } from '../../chat';

const ChatHistory = () => {
  const { conversations, fetchMessages } = useChat();

  return (
    <Box sx={{ padding: 2 }}>
      <Typography variant="h6" gutterBottom>
        Chat History
      </Typography>
      {(conversations?.length === 0 || !conversations) ? (
        <Typography variant="body2" color="textSecondary">
          No chat history available.
        </Typography>
      ) : (
        conversations?.map((chat, index) => (
          <ChatItem key={index} chatId={chat.id ?? 0} chatTitle={chat.title ?? 'Untitled Chat'} onSelect={fetchMessages} />
        ))
      )}
    </Box>
  );
};

export default ChatHistory;