import { CircularProgress, Typography } from '@mui/material';

interface ControlLoaderProps {
  text?: string;
}

const ControlLoader: React.FC<ControlLoaderProps> = ({text}) => {
  return (
    <div style={{ display: 'flex', alignItems: 'center', padding: '10px' }}>
      <CircularProgress size={24} />
      <Typography variant="body2" style={{ marginLeft: '8px' }}>
        {text || 'Loading...'}
      </Typography>
    </div>
  );
};

export default ControlLoader;