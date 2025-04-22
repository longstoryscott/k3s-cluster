import React from 'react';
import { Box, Button, Typography } from '@mui/material';

const models = [
  { id: 'gemma3:1b', name: 'Gemma 3.1B' },
  { id: 'gemma3:2b', name: 'Gemma 3.2B' },
  { id: 'gemma3:3b', name: 'Gemma 3.3B' }
  // Add more models as needed
];

interface ModelSelectorProps {
  onSelect: (modelId: string) => void;
}

const ModelSelector: React.FC<ModelSelectorProps> = ({ onSelect }) => {
  return (
    <Box>
      <Typography variant="h6" gutterBottom>
        Select a Model
      </Typography>
      <Box display="flex" flexDirection="column" gap={2}>
        {models.map((model) => (
          <Button
            key={model.id}
            variant="outlined"
            onClick={() => onSelect(model.id)}
          >
            {model.name}
          </Button>
        ))}
      </Box>
    </Box>
  );
};

export default ModelSelector;