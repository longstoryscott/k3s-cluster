import React from 'react';
import { Button, IconButton, Tooltip } from '@mui/material';
import { Delete, Edit, Reply } from '@mui/icons-material';

interface MessageActionsProps {
  onEdit: () => void;
  onDelete: () => void;
  onReply: () => void;
}

const MessageActions: React.FC<MessageActionsProps> = ({ onEdit, onDelete, onReply }) => {
  return (
    <div style={{ display: 'flex', gap: '8px' }}>
      <Tooltip title="Edit">
        <IconButton onClick={onEdit}>
          <Edit />
        </IconButton>
      </Tooltip>
      <Tooltip title="Delete">
        <IconButton onClick={onDelete}>
          <Delete />
        </IconButton>
      </Tooltip>
      <Tooltip title="Reply">
        <Button variant="contained" onClick={onReply} startIcon={<Reply />}>
          Reply
        </Button>
      </Tooltip>
    </div>
  );
};

export default MessageActions;