import { useState, useEffect } from 'react';
import { Box, Typography, Button, Alert, Grid, FormControl, Select, MenuItem, InputLabel, SelectChangeEvent } from '@mui/material';
import { useConfigContext } from '../../context/ConfigContext';
import { ModelProfile, ModelProfilesConfig } from '../../hooks/useConfig';
import { useAuth } from '../../auth';
import { listModelProfiles } from '../../api/model';
import { updateConfig } from '../../api';

const TASKS: { key: keyof ModelProfilesConfig; label: string; }[] = [
  { key: 'primaryProfileId', label: 'Primary' },
  { key: 'primarySummaryProfileId', label: 'Summary' },
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

const ModelProfileSelector = ({ task, profiles }: { task: { key: keyof ModelProfilesConfig; label: string; }; profiles: ModelProfile[] }) => {
  const { config, updateConfig } = useConfigContext();
  const [value, setValue] = useState(config?.modelProfiles?.[task.key] || '');
  
  // Use effect to update value when config changes
  useEffect(() => {
    if (config?.modelProfiles && task.key in config.modelProfiles) {
      setValue(config.modelProfiles[task.key] || '');
    }
  }, [config, task.key]);
  
  const handleChange = (event: SelectChangeEvent) => {
    const newValue = event.target.value as string;
    setValue(newValue);
    if (config?.modelProfiles) {
      updateConfig({
        ...config,
        modelProfiles: {
          ...config.modelProfiles,
          [task.key]: newValue
        }
      });
    }
  };
  return (
    <Grid key={task.key}>
      <FormControl fullWidth>
        <InputLabel>{task.label}</InputLabel>
        <Select
          value={value}
          onChange={handleChange}
          labelId={`${task.key}-select-label`}
          id={`${task.key}-select`}
          label={task.label}
        >
          <MenuItem value="">(None)</MenuItem>
          {profiles && profiles.map(profile => (
            <MenuItem key={profile.id} value={profile.id}>{profile.name}</MenuItem>
          ))}
        </Select>
      </FormControl>
    </Grid>
  )
}

const ModelSettings = () => {
  const { config, isLoading } = useConfigContext();
  const [profiles, setProfiles] = useState<ModelProfile[]>([]);
  const [saveStatus, setSaveStatus] = useState<{success?: boolean; message: string} | null>(null);
  const auth = useAuth();
  const [isSaving, setIsSaving] = useState(false);

  // Fetch profiles on mount
  useEffect(() => {
    const fetchProfiles = async () => {
      try {
        // You may need to pass the token here
        const data = await listModelProfiles(auth.user.accessToken || '');
        setProfiles(data);
      } catch (err: unknown) {
        if (err instanceof Error) {
          console.error('Error fetching model profiles:', err.message);
        }
      }
    };
    fetchProfiles();
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

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

      const success = await updateConfig(auth.user.accessToken || '', config)
      
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
          {TASKS.map(task => (<ModelProfileSelector key={task.key} task={task} profiles={profiles} />))}
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