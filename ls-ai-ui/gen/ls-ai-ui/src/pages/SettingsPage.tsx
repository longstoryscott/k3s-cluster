import React from 'react';
import { Grid, Typography } from '@mui/material';
import ApiSettings from '../components/Settings/ApiSettings';
import ProfileSettings from '../components/Settings/ProfileSettings';

const SettingsPage = () => {
  return (
    <Grid container spacing={3} style={{ padding: 20 }}>
      <Grid item xs={12}>
        <Typography variant="h4" gutterBottom>
          Settings
        </Typography>
      </Grid>
      <Grid item xs={12} md={6}>
        <ApiSettings />
      </Grid>
      <Grid item xs={12} md={6}>
        <ProfileSettings />
      </Grid>
    </Grid>
  );
};

export default SettingsPage;