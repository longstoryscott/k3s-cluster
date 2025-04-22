import React from 'react';
import { Card, CardContent, Typography, Button } from '@mui/material';

interface ModelCardProps {
  modelName: string;
  modelDescription: string;
  onSelect: (model: string) => void;
}

const ModelCard: React.FC<ModelCardProps> = ({ modelName, modelDescription, onSelect }) => {
  return (
    <Card variant="outlined" style={{ margin: '10px', cursor: 'pointer' }}>
      <CardContent>
        <Typography variant="h5" component="div">
          {modelName}
        </Typography>
        <Typography variant="body2" color="text.secondary">
          {modelDescription}
        </Typography>
        <Button variant="contained" color="primary" onClick={() => onSelect(modelName)}>
          Select Model
        </Button>
      </CardContent>
    </Card>
  );
};

export default ModelCard;