import React from 'react';
import { Box, FormControl, InputLabel, Select, MenuItem, Typography } from '@mui/material';
import { SelectChangeEvent } from '@mui/material/Select';
import { useChat } from '../../chat';
import ControlLoader from '../Shared/ControlLoader';

const ModelSelector: React.FC = () => {
  const { selectedModel, setSelectedModel, models, isLoading } = useChat();

  const handleModelChange = (event: SelectChangeEvent) => {
    const modelId = event.target.value as string;
    setSelectedModel(modelId);
  };

  return (
    isLoading ? 
      <ControlLoader text='Loading models...' /> :
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
            {models && models?.map((model) => (
              <MenuItem key={model.name} value={model.name}>
                {model.name}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
      </Box>
  );
};

export default ModelSelector;