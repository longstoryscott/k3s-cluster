import { useEffect, useRef } from 'react';
import ChatHeader from './ChatHeader';
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

  // A proper handler function instead of just logging to console
  const handleMenuClick = () => {
    // This could be implemented to toggle a drawer or handle other UI navigation
    // For now, this is a placeholder for future functionality
  };

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
      <ChatHeader title="Chat" onMenuClick={handleMenuClick} />
      <Box sx={{ flex: 1, p: 2 }}>
        {/* Children will contain the messages displayed by ChatPage */}
        {children}
      </Box>
      <ChatInput />
    </Box>
  );
};

export default ChatContainer;