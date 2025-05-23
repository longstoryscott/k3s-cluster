import { Box } from '@mui/material';
import { memo, useEffect, useRef, useLayoutEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import ChatContainer from '../components/Chat/ChatContainer';
import ChatBubble from '../components/Chat/ChatBubble';
import { useChat } from '../chat';
import ControlLoader from '../components/Shared/ControlLoader';
import { ChatMessage } from '../api';

const ChatPage = memo(() => {
  const { messages, response, isTyping, currentConversation, selectConversation } = useChat();
  const { conversationId } = useParams();
  const containerRef = useRef<HTMLBodyElement>(document.body as HTMLBodyElement);
  const shouldScrollToBottom = useRef<boolean>(true);
  const lastScrollTime = useRef<number>(0);
  const [currentMessage, setCurrentMessage] = useState<ChatMessage>({
    role: 'assistant' as const,
    content: response,
    id: (messages[messages.length - 1]?.id ?? 0) + 1
  });


  // Throttle scroll events to improve performance
  const handleScroll = () => {
    const now = Date.now();
    // Only process scroll events every 100ms
    if (now - lastScrollTime.current > 100) {
      // If user scrolls up more than 10px from bottom, disable auto-scrolling
      const isAtBottom = containerRef.current.scrollHeight - (window.scrollY + window.innerHeight) < 100;
      shouldScrollToBottom.current = isAtBottom;
      lastScrollTime.current = now;
    }
  };

  // Load conversation from URL parameter when component mounts or conversationId changes
  useEffect(() => {
    if (conversationId) {
      const numericId = parseInt(conversationId, 10);
      if (!isNaN(numericId)) {
        // Only call selectConversation if the conversationId is different from the currentConversation.id
        if (!currentConversation || currentConversation.id !== numericId) {
          selectConversation(numericId);
          shouldScrollToBottom.current = true;
        }
      }
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [conversationId, currentConversation]);
  
  // Track user scroll position
  useEffect(() => {
    window.addEventListener('scroll', handleScroll);
    return () => window.removeEventListener('scroll', handleScroll);
  }, []);
  
  // Scroll to bottom whenever messages change or streaming occurs - with priority timing
  useLayoutEffect(() => {
    if (!shouldScrollToBottom.current) {
      return;
    }
    
    const scrollToBottom = () => {
      if (containerRef.current) {
        window.scrollTo(0, containerRef.current.scrollHeight);
      }
    };
    
    // Immediate scroll
    scrollToBottom();
    
    // Additional scroll after a short delay to ensure content is rendered
    const timeoutId = setTimeout(() => {
      scrollToBottom();
    }, 50);
    
    return () => clearTimeout(timeoutId);
  }, [messages, response, isTyping]);

  useEffect(() => {
    setCurrentMessage(prev => ({
      ...prev,
      content: response
    }));
  }, [response]);

  return (
    <Box 
      sx={{ 
        display: 'flex', 
        flexDirection: 'column', 
        height: '100%'
      }} 
    >
      <ChatContainer>
        {/* Display all existing messages */}
        {messages.map((msg, index) => (
          <ChatBubble 
            key={`msg-${index}`} 
            message={msg} 
          />
        ))}
          
        {/* Only display in-progress response if it's not already in messages */}
        {currentMessage.content && (
          <ChatBubble 
            key="streaming-response"
            message={currentMessage} 
            inProgress={isTyping} 
          />
        )}
          
        {/* Display typing indicator when no response content yet */}
        {isTyping && (
          <ControlLoader text={response ? 'Writing...' : 'Thinking...'} />
        )}
      </ChatContainer>
    </Box>
  );
});

export default ChatPage;