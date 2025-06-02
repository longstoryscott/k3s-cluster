import React, { useState } from 'react';
import { TextField, Button, Box, Typography, useTheme, Switch, FormControlLabel } from '@mui/material';
import { useChat } from '../../chat';
import { useConfigContext } from '../../context/ConfigContext';
import SearchIndicator from './SearchIndicator';

const ChatInput = () => {
  const [input, setInput] = useState('');
  const { sendMessage, isTyping, currentConversation, isSearching } = useChat();
  const theme = useTheme();
  const { config, updatePartialConfig, isLoading } = useConfigContext();
  const alwaysRetrieve = config?.retrieval?.alwaysRetrieve || false;

  // Check if there's an active conversation
  const hasConversation = !!currentConversation?.id;

  // Get toggle states from config (default to false if not loaded)
  const summarizationEnabled = config?.summarization?.enabled !== false;
  const critiqueEnabled = config?.summarization?.enableResponseCritique || false;
  const retrievalEnabled = config?.retrieval?.enabled !== false;
  const webSearchEnabled = config?.webSearch?.enabled || false;

  // Handlers for toggles
  const handleToggleSummarization = async () => {
    await updatePartialConfig('summarization', {
      ...config?.summarization,
      enabled: !summarizationEnabled
    });
  };
  const handleToggleCritique = async () => {
    await updatePartialConfig('summarization', {
      ...config?.summarization,
      enableResponseCritique: !critiqueEnabled
    });
  };
  const handleToggleRetrieval = async () => {
    await updatePartialConfig('retrieval', {
      ...config?.retrieval,
      enabled: !retrievalEnabled
    });
  };
  const handleToggleAlwaysRetrieve = async () => {
    await updatePartialConfig('retrieval', {
      ...config?.retrieval,
      alwaysRetrieve: !alwaysRetrieve
    });
  };
  
  const handleToggleWebSearch = async () => {
    await updatePartialConfig('webSearch', {
      ...config?.webSearch,
      enabled: !webSearchEnabled
    });
  };

  const handleKeyPress = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'Enter' && !event.shiftKey) {
      event.preventDefault();
      handleSend();
    }
  };

  const handleSend = () => {
    const trimmedInput = input.trim();
    if (trimmedInput && !isTyping && hasConversation) {
      sendMessage({ content: trimmedInput, role: 'user', conversationId: currentConversation.id! });
      setInput('');
    }
  };

  return (
    <Box 
      sx={{ 
        display: 'flex', 
        flexDirection: 'column',
        p: theme.spacing(2),
        borderTop: `${theme.spacing(0.125)} solid`,
        borderColor: theme.palette.divider
      }}
    >
      {/* Search indicator when web search is in progress */}
      {isSearching && <SearchIndicator />}
      
      {/* Quick toggles for summarization, critique, and retrieval */}
      <Box sx={{ display: 'flex', alignItems: 'center', mb: theme.spacing(1), gap: 2, flexWrap: 'wrap' }}>
        <FormControlLabel
          control={
            <Switch
              checked={summarizationEnabled}
              onChange={handleToggleSummarization}
              disabled={isLoading}
              color="primary"
            />
          }
          label="Summarization"
        />
        <FormControlLabel
          control={
            <Switch
              checked={critiqueEnabled}
              onChange={handleToggleCritique}
              disabled={isLoading}
              color="primary"
            />
          }
          label="Critique"
        />
        <FormControlLabel
          control={
            <Switch
              checked={retrievalEnabled}
              onChange={handleToggleRetrieval}
              disabled={isLoading}
              color="primary"
            />
          }
          label="Memory Retrieval"
        />
        <FormControlLabel
          control={
            <Switch
              checked={alwaysRetrieve}
              onChange={handleToggleAlwaysRetrieve}
              disabled={isLoading}
              color="primary"
            />
          }
          label="Always Enable Memory Retrieval"
        />
        <FormControlLabel
          control={
            <Switch
              checked={webSearchEnabled}
              onChange={handleToggleWebSearch}
              disabled={isLoading}
              color="primary"
            />
          }
          label="Web Search"
        />
      </Box>
      {!hasConversation && (
        <Typography 
          variant="body2" 
          color="text.secondary" 
          sx={{ mb: theme.spacing(1), textAlign: 'center' }}
        >
          Start a new conversation to begin chatting
        </Typography>
      )}
      <Box sx={{ display: 'flex', alignItems: 'center' }}>
        <TextField
          variant="outlined"
          fullWidth
          placeholder={hasConversation ? "Type your message..." : "No active conversation..."}
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={handleKeyPress}
          multiline
          maxRows={4}
          disabled={isTyping || !hasConversation}
        />
        <Button 
          onClick={handleSend} 
          variant="contained" 
          color="primary" 
          sx={{ ml: theme.spacing(2) }}
          disabled={!input.trim() || isTyping || !hasConversation}
          type='submit'
        >
          Send
        </Button>
      </Box>
    </Box>
  );
};

export default ChatInput;