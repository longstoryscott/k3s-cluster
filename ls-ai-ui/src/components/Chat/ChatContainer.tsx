import { useEffect, useRef } from 'react';
import ChatHeader from './ChatHeader';
import ChatInput from './ChatInput';
import ChatBubble from './ChatBubble';
import TypingIndicator from './TypingIndicator';
import { Box } from '@mui/material';
import { useChat } from '../../chat';

const ChatContainer = ({children}: React.PropsWithChildren<unknown>) => {
  const { response, isTyping, messages } = useChat();
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    // Scroll to the bottom of the chat when messages change
    if (containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight;
    }
  }, [response]);

  return (
    <Box 
      ref={containerRef}
      sx={{ 
        height: '100%', 
        display: 'flex',
        flexDirection: 'column',
        overflow: 'auto'
      }}
    >
      <ChatHeader title="Chat" onMenuClick={() => console.log('Menu clicked')} />
      <Box sx={{ flex: 1, p: 2 }}>
        {children}
        {messages.map((msg, index) => (
          <ChatBubble key={index} message={msg} />
        ))}
        {isTyping && <TypingIndicator />}
      </Box>
      <ChatInput />
    </Box>
  );
};

export default ChatContainer;