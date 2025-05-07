import { Box, CircularProgress, Typography, useTheme } from '@mui/material';

interface ControlLoaderProps {
  text?: string;
}

const ControlLoader = ({ text = 'Loading...' }: ControlLoaderProps) => {
  const theme = useTheme();
  
  return (
    <Box 
      sx={{ 
        display: 'flex', 
        flexDirection: 'column',
        justifyContent: 'center', 
        alignItems: 'center', 
        p: theme.spacing(2)
      }}
    >
      <CircularProgress size={24} />
      {text && (
        <Typography 
          variant="body2" 
          color="text.secondary" 
          sx={{ mt: theme.spacing(1) }}
        >
          {text}
        </Typography>
      )}
    </Box>
  );
};

export default ControlLoader;