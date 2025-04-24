import { Box } from '@mui/material';
import ChatContainer from '../components/Chat/ChatContainer';
import ChatBubble from '../components/Chat/ChatBubble';
import ModelSelector from '../components/ModelSelector/ModelSelector';
import { useChat } from '../chat';
import ControlLoader from '../components/Shared/ControlLoader';

const ChatPage = () => {
  const { messages, response, isTyping } = useChat();

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column' }}>
      <ModelSelector />
      <Box sx={{ flex: 1, overflow: 'hidden' }}>
        <ChatContainer>
          {/* Display all existing messages */}
          {messages.map((msg, index) => (
            <ChatBubble key={`msg-${index}`} message={msg} />
          ))}
          
          {/* Only display in-progress response if it's not already in messages */}
          {response && isTyping && (
            <ChatBubble 
              key="streaming-response"
              message={{ role: 'assistant', content: response }} 
              inProgress={true} 
            />
          )}
          
          {/* Display typing indicator when no response content yet */}
          {isTyping && !response && <ControlLoader text='Typing...'/>}
        </ChatContainer>
      </Box>
    </Box>
  );
};

export default ChatPage;