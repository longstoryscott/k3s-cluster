import { Button } from '@mui/material';
import { useChat } from '../../chat';
import { useNavigate } from 'react-router-dom';

const NewChatButton = () => {
  const { startNewConversation } = useChat();
  const navigate = useNavigate();

  const handleNewChat = () => {
    startNewConversation();
    navigate('/'); // Navigate to root, ChatPage will update URL once conversation is created
  };

  return (
    <Button 
      variant="contained" 
      color="primary" 
      onClick={handleNewChat} 
      fullWidth
    >
      New Chat
    </Button>
  );
};

export default NewChatButton;