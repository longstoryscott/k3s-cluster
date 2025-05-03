import { Box, Button } from '@mui/material';
import ChatContainer from '../components/Chat/ChatContainer';
import ChatBubble from '../components/Chat/ChatBubble';
import { useChat } from '../chat';
import ControlLoader from '../components/Shared/ControlLoader';
import SummarizeIcon from '@mui/icons-material/Summarize';
import { useState } from 'react';
import { useAuth } from '../auth';
import config from '../config';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';

const ChatPage = () => {
  const { messages, response, isTyping, currentConversation } = useChat();
  const [isSummarizing, setIsSummarizing] = useState(false);
  const [summary, setSummary] = useState<string | null>(null);
  const { user } = useAuth();

  const handleSummarize = async () => {
    if (!currentConversation?.id || isSummarizing) {
      return;
    }

    setIsSummarizing(true);
    try {
      const response = await fetch(
        `${config.server.baseUrl}/api/conversations/${currentConversation.id}/summarize`, 
        {
          headers: {
            'Authorization': `Bearer ${user.accessToken}`,
            'Content-Type': 'application/json'
          }
        }
      );

      if (response.ok) {
        const data = await response.json();
        setSummary(data.content || "Conversation summarized successfully.");
      } else {
        console.error('Failed to summarize conversation');
        setSummary("Failed to summarize conversation.");
      }
    } catch (error) {
      console.error('Error summarizing conversation:', error);
      setSummary("Error summarizing conversation.");
    } finally {
      setIsSummarizing(false);
    }
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <Box sx={{ flex: 1, overflow: 'hidden' }}>
        <ChatContainer>
          {/* Display all existing messages */}
          {messages.map((msg, index) => (
            <ChatBubble key={`msg-${index}`} message={msg} />
          ))}
          
          {/* Display summary if available */}
          {summary && (
            <Box 
              sx={{ 
                padding: 2, 
                margin: 2, 
                bgcolor: 'background.paper', 
                borderRadius: 2,
                border: '1px dashed',
                borderColor: 'primary.main'
              }}
            >
              <Box sx={{ fontWeight: 'bold', mb: 1 }} className="markdown-body">Conversation Summary:</Box>
              <ReactMarkdown remarkPlugins={[remarkGfm]}>
                {summary}
              </ReactMarkdown>
            </Box>
          )}
          
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
          
          {/* Summarize button */}
          {currentConversation?.id && messages.length > 5 && (
            <Box sx={{ display: 'flex', justifyContent: 'center', my: 2 }}>
              <Button
                variant="outlined"
                color="primary"
                startIcon={<SummarizeIcon />}
                onClick={handleSummarize}
                disabled={isSummarizing}
              >
                {isSummarizing ? 'Summarizing...' : 'Summarize Conversation'}
              </Button>
            </Box>
          )}
        </ChatContainer>
      </Box>
    </Box>
  );
};

export default ChatPage;