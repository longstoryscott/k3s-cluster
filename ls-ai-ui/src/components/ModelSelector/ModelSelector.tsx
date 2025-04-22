import React from 'react';
import { Box, FormControl, InputLabel, Select, MenuItem, Typography } from '@mui/material';
import { SelectChangeEvent } from '@mui/material/Select';
import { useChat } from '../../chat';

const models = [
  { id: 'llama3-8b', name: 'Llama 3 8B (Fast)' },
  { id: 'phi3.5', name: 'Phi 3.5' }
];

const ModelSelector: React.FC = () => {
  const { selectedModel, setSelectedModel } = useChat();

  const handleModelChange = (event: SelectChangeEvent) => {
    const modelId = event.target.value as string;
    setSelectedModel(modelId);
  };
  

  return (
    <Box sx={{ mb: 2, p: 2 }}>
      <Typography variant="h6" gutterBottom>
        Select a Model
      </Typography>
      <FormControl fullWidth>
        <InputLabel id="model-select-label">Model</InputLabel>
        <Select
          labelId="model-select-label"
          id="model-select"
          value={selectedModel}
          onChange={handleModelChange}
          label="Model"
        >
          {models.map((model) => (
            <MenuItem key={model.id} value={model.id}>
              {model.name}
            </MenuItem>
          ))}
        </Select>
      </FormControl>
    </Box>
  );
};

export default ModelSelector;