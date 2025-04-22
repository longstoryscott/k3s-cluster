import { Box, Paper } from '@mui/material';
import ChatContainer from '../components/Chat/ChatContainer';
import ChatBubble from '../components/Chat/ChatBubble';
import ModelSelector from '../components/ModelSelector/ModelSelector';
import { useChat } from '../chat';
import TypingIndicator from '../components/Chat/TypingIndicator';

const ChatPage = () => {
  const { messages, response, isTyping } = useChat();

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100vh' }}>
      <ModelSelector />
      <Paper elevation={2} sx={{ flex: 1, overflow: 'auto', p: 2 }}>
        <ChatContainer>
          {/* Display all existing messages */}
          {messages.map((msg, index) => (
            <ChatBubble key={`msg-${index}`} message={msg} />
          ))}
          
          {/* Display in-progress response */}
          {response && isTyping && (
            <ChatBubble 
              message={{ role: 'assistant', content: response }} 
              inProgress={true} 
            />
          )}
          
          {/* Display typing indicator */}
          {isTyping && !response && <TypingIndicator />}
        </ChatContainer>
      </Paper>
    </Box>
  );
};

export default ChatPage;