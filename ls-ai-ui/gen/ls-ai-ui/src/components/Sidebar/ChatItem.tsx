import React from 'react';
import { ListItem, ListItemText, ListItemIcon, IconButton } from '@mui/material';
import DeleteIcon from '@mui/icons-material/Delete';
import { useChat } from '../../hooks/useChat';

interface ChatItemProps {
  chatId: string;
  chatTitle: string;
  onSelect: (chatId: string) => void;
}

const ChatItem: React.FC<ChatItemProps> = ({ chatId, chatTitle, onSelect }) => {
  const { deleteChat } = useChat();

  const handleDelete = (event: React.MouseEvent) => {
    event.stopPropagation();
    deleteChat(chatId);
  };

  return (
    <ListItem button onClick={() => onSelect(chatId)}>
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