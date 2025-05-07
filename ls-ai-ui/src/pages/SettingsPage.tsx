import React from 'react';
import { Grid, Typography, useTheme } from '@mui/material';
import ApiSettings from '../components/Settings/ApiSettings';
import ProfileSettings from '../components/Settings/ProfileSettings';
import ModelSettings from '../components/Settings/ModelSettings';

const SettingsPage = () => {
  const theme = useTheme();
  
  return (
    <Grid container spacing={3} sx={{ padding: theme.spacing(2.5) }}>
      <Grid size={12}>
        <Typography variant="h4" gutterBottom>
          Settings
        </Typography>
      </Grid>
      <Grid size={12}>
        <ModelSettings />
      </Grid>
      <Grid size={12}>
        <ApiSettings />
      </Grid>
      <Grid size={12}>
        <ProfileSettings />
      </Grid>
    </Grid>
  );
};

export default SettingsPage;