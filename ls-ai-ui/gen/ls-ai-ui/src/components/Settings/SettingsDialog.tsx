import React, { useState } from 'react';
import { Dialog, DialogTitle, DialogContent, DialogActions, Button, TextField, Typography } from '@mui/material';
import { useAuth } from '../../auth/AuthProvider';

const SettingsDialog = ({ open, onClose }) => {
  const { user, logout } = useAuth();
  const [username, setUsername] = useState(user.name);
  const [email, setEmail] = useState(user.email);

  const handleSave = () => {
    // Logic to save settings (e.g., update user profile)
    console.log('Settings saved:', { username, email });
    onClose();
  };

  const handleLogout = () => {
    logout();
    onClose();
  };

  return (
    <Dialog open={open} onClose={onClose}>
      <DialogTitle>Settings</DialogTitle>
      <DialogContent>
        <Typography variant="h6">Profile Settings</Typography>
        <TextField
          label="Username"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          fullWidth
          margin="normal"
        />
        <TextField
          label="Email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          fullWidth
          margin="normal"
        />
      </DialogContent>
      <DialogActions>
        <Button onClick={handleLogout} color="secondary">Logout</Button>
        <Button onClick={handleSave} color="primary">Save</Button>
      </DialogActions>
    </Dialog>
  );
};

export default SettingsDialog;