import { memo } from 'react';
import ChatInput from './ChatInput';
import { Box, useTheme } from '@mui/material';

const ChatContainer = memo(({children}: React.PropsWithChildren<unknown>) => {
  const theme = useTheme();

  return (
    <>
      <Box sx={{ flex: 1, p: theme.spacing(2), overflow: 'auto' }}>
        {/* Children will contain the messages displayed by ChatPage */}
        {children}
      </Box>
      <ChatInput />
    </>
  );
});

export default ChatContainer;