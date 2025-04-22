import React from 'react';
import { ListItem, ListItemText, ListItemIcon, IconButton } from '@mui/material';
import DeleteIcon from '@mui/icons-material/Delete';
import { useChat } from '../../chat';

interface ChatItemProps {
  chatId: number;
  chatTitle: string;
  onSelect: (chatId: number) => void;
}

const ChatItem: React.FC<ChatItemProps> = ({ chatId, chatTitle, onSelect }) => {
  const { deleteConversation } = useChat();

  const handleDelete = (event: React.MouseEvent) => {
    event.stopPropagation();
    deleteConversation(chatId);
  };

  return (
    <ListItem sx={{ cursor: 'pointer' }} onClick={() => onSelect(chatId)}>
      <ListItemText primary={chatTitle} />
      <ListItemIcon>
        <IconButton edge="end" aria-label="delete" onClick={handleDelete}>
          <DeleteIcon />
        </IconButton>
      </ListItemIcon>
    </ListItem>
  );
};

export default ChatItem;