import { useState } from 'react';
import { Box, Typography, Button, Alert, Grid } from '@mui/material';
import { useConfigContext } from '../../context/ConfigContext';
import { ModelProfilesConfig } from '../../hooks/useConfig';
import { useAuth } from '../../auth';
import { getToken, updateConfig } from '../../api';
import ModelProfileSelector from '../ModelSelector/ModelProfileSelector';

const TASKS: { key: keyof ModelProfilesConfig; label: string; }[] = [
  { key: 'primaryProfileId', label: 'Primary' },
  { key: 'summarizationProfileId', label: 'Summary' },
  { key: 'masterSummaryProfileId', label: 'Master Summary' },
  { key: 'briefSummaryProfileId', label: 'Brief Summary' },
  { key: 'keyPointsProfileId', label: 'Key Points' },
  { key: 'selfCritiqueProfileId', label: 'Self Critique' },
  { key: 'improvementProfileId', label: 'Improvement' },
  { key: 'memoryRetrievalProfileId', label: 'Memory Retrieval' },
  { key: 'analysisProfileId', label: 'Analysis' },
  { key: 'researchTaskProfileId', label: 'Research Task' },
  { key: 'researchPlanProfileId', label: 'Research Plan' },
  { key: 'researchConsolidationProfileId', label: 'Research Consolidation' },
  { key: 'researchAnalysisProfileId', label: 'Research Analysis' },
  { key: 'embeddingProfileId', label: 'Embeddings' }
];

const ModelSettings = () => {
  const { config, isLoading } = useConfigContext();
  const [saveStatus, setSaveStatus] = useState<{success?: boolean; message: string} | null>(null);
  const auth = useAuth();
  const [isSaving, setIsSaving] = useState(false);

  const handleSave = async () => {
    setSaveStatus(null);
    setIsSaving(true);
    
    try {
      if (!config) {
        setSaveStatus({
          success: false,
          message: 'No configuration available to save.'
        });
        return;
      }

      console.log('Saving config:', config);

      const success = await updateConfig(getToken(auth.user), config)
      
      if (success) {
        setSaveStatus({
          success: true,
          message: 'Model settings saved successfully!'
        });
      } else {
        setSaveStatus({
          success: false,
          message: 'Failed to save settings.'
        });
      }
    } catch (err) {
      setSaveStatus({
        success: false,
        message: `Error: ${err instanceof Error ? err.message : String(err)}`
      });
    } finally {
      setIsSaving(false);
    }
  };

  if (isLoading) {
    return <Box sx={{ padding: 2 }}><Typography>Loading model settings...</Typography></Box>;
  }

  return (
    <Box sx={{ padding: 2 }}>
      <Typography variant="h6" gutterBottom>
        Model Settings
      </Typography>
      
      {saveStatus && (
        <Alert 
          severity={saveStatus.success ? "success" : "error"} 
          sx={{ mb: 2 }}
          onClose={() => setSaveStatus(null)}
        >
          {saveStatus.message}
        </Alert>
      )}
      
      <Box sx={{ mt: 4 }}>
        <Typography variant="h6">Assign Profiles to Tasks</Typography>
        <Grid container spacing={2} sx={{ p: 2 }}>
          {TASKS.map(task => (<ModelProfileSelector key={task.key} task={task} />))}
        </Grid>
      </Box>
      
      <Button 
        variant="contained" 
        color="primary" 
        sx={{ mt: 2 }} 
        onClick={handleSave}
        disabled={isSaving}
      >
        {isSaving ? 'Saving...' : 'Save Model Settings'}
      </Button>
    </Box>
  );
};

export default ModelSettings;