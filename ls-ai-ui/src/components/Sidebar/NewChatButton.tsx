import { Button } from '@mui/material';
import { useChat } from '../../chat';

const NewChatButton = () => {
  const { startNewConversation } = useChat();

  const handleNewChat = () => {
    startNewConversation();
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