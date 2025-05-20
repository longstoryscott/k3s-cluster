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
  const [currentMessage, setCurrentMessage] = useState<ChatMessage>({
    role: 'assistant' as const,
    content: response,
    id: (messages[messages.length - 1]?.id ?? 0) + 1
  });

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

  // // Update URL when currentConversation changes
  // useEffect(() => {
  //   if (currentConversation?.id && (!conversationId || parseInt(conversationId, 10) !== currentConversation.id)) {
  //     navigate(`/chat/${currentConversation.id}`, { replace: true });
  //   }
  // }, [currentConversation?.id, conversationId]);
  
  // Track user scroll position
  useEffect(() => {
    const container = containerRef.current;
    if (!container) {
      return;
    }
    
    const handleScroll = () => {
      // If user scrolls up more than 100px from bottom, disable auto-scrolling
      const isAtBottom = container.scrollHeight - (window.scrollY + window.screen.height) < 100;
      shouldScrollToBottom.current = isAtBottom;
    };
    
    window.addEventListener('scroll', handleScroll);
    return () => window.removeEventListener('scroll', handleScroll);
  }, [containerRef]);
  
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

  // Prepare the streaming response - create a stable object that won't cause unnecessary re-renders
  // const streamingMessage = response ? {
  //   role: 'assistant' as const,
  //   content: response,
  //   id: (messages[messages.length - 1]?.id ?? 0) + 1
  // } : null;

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
          <ControlLoader text='Typing...'/>
        )}
      </ChatContainer>
    </Box>
  );
});

export default ChatPage;