import { useEffect, useRef } from 'react';
import ChatInput from './ChatInput';
import { Box } from '@mui/material';
import { useChat } from '../../chat';

const ChatContainer = ({children}: React.PropsWithChildren<unknown>) => {
  const { response, isTyping } = useChat();
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    // Scroll to the bottom of the chat when messages change or during typing
    if (containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight;
    }
  }, [response, isTyping]);

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
      <Box sx={{ flex: 1, p: 2, overflow: 'auto' }}>
        {/* Children will contain the messages displayed by ChatPage */}
        {children}
      </Box>
      <ChatInput />
    </Box>
  );
};

export default ChatContainer;