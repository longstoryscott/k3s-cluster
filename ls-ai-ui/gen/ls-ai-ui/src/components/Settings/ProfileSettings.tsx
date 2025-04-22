import React, { useState } from 'react';
import { Button, TextField, Typography, Paper, Grid } from '@mui/material';
import { useAuth } from '../../auth/AuthProvider';

const ProfileSettings = () => {
  const { user, logout } = useAuth();
  const [name, setName] = useState(user.name);
  const [email, setEmail] = useState(user.email);

  const handleSave = () => {
    // Logic to save the updated profile information
    console.log('Profile updated:', { name, email });
  };

  return (
    <Paper elevation={3} style={{ padding: 20 }}>
      <Typography variant="h5" gutterBottom>
        Profile Settings
      </Typography>
      <Grid container spacing={2}>
        <Grid item xs={12}>
          <TextField
            label="Name"
            variant="outlined"
            fullWidth
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </Grid>
        <Grid item xs={12}>
          <TextField
            label="Email"
            variant="outlined"
            fullWidth
            value={email}
            onChange={(e) => setEmail(e.target.value)}
          />
        </Grid>
        <Grid item xs={12}>
          <Button variant="contained" color="primary" onClick={handleSave}>
            Save Changes
          </Button>
          <Button variant="outlined" color="secondary" onClick={logout} style={{ marginLeft: 10 }}>
            Logout
          </Button>
        </Grid>
      </Grid>
    </Paper>
  );
};

export default ProfileSettings;