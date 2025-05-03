import { useState, useEffect } from 'react';
import { Box, FormControl, InputLabel, Select, MenuItem, Typography, Button } from '@mui/material';
import { SelectChangeEvent } from '@mui/material/Select';
import { useChat } from '../../chat';
import ControlLoader from '../Shared/ControlLoader';

const ModelSettings = () => {
  const { selectedModel, setSelectedModel, models, isLoading, fetchModels } = useChat();
  const [modelDescription, setModelDescription] = useState('');

  useEffect(() => {
    // Refresh the models list when component mounts
    fetchModels();
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleModelChange = (event: SelectChangeEvent) => {
    const modelId = event.target.value as string;
    setSelectedModel(modelId);
    
    // Find the selected model and show its description (if we had descriptions)
    const model = models?.find(m => m.name === modelId);
    setModelDescription(model?.details?.family || '');
  };

  return (
    <Box sx={{ padding: 2 }}>
      <Typography variant="h6" gutterBottom>
        Model Settings
      </Typography>
      
      {isLoading ? (
        <ControlLoader text='Loading models...' />
      ) : (
        <>
          <FormControl fullWidth margin="normal">
            <InputLabel id="model-select-label">Default Model</InputLabel>
            <Select
              labelId="model-select-label"
              id="model-select"
              value={selectedModel}
              onChange={handleModelChange}
              label="Default Model"
            >
              {models && models?.map((model) => (
                <MenuItem key={model.name} value={model.name}>
                  {model.name}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
          
          {modelDescription && (
            <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
              Model family: {modelDescription}
            </Typography>
          )}
          
          <Button 
            variant="contained" 
            color="primary" 
            sx={{ mt: 2 }}
            onClick={fetchModels}
            disabled={isLoading}
          >
            Refresh Models
          </Button>
        </>
      )}
    </Box>
  );
};

export default ModelSettings;