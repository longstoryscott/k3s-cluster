import { useState } from 'react';
import { Box, Button, TextField, Typography } from '@mui/material';
import config from '../../config';

const ApiSettings = () => {
  const [apiUrl, setApiUrl] = useState(config.server.baseUrl);
  const [clientId, setClientId] = useState(config.auth.clientId);
  const [clientSecret, setClientSecret] = useState(config.auth.clientSecret);

  const handleSave = () => {
    // Save the API settings to local storage or send to the server
    localStorage.setItem('apiUrl', apiUrl);
    localStorage.setItem('clientId', clientId);
    localStorage.setItem('clientSecret', clientSecret);
    alert('API settings saved successfully!');
  };

  return (
    <Box sx={{ padding: 2 }}>
      <Typography variant="h6">API Settings</Typography>
      <TextField
        label="API URL"
        value={apiUrl}
        onChange={(e) => setApiUrl(e.target.value)}
        fullWidth
        margin="normal"
      />
      <TextField
        label="Client ID"
        value={clientId}
        onChange={(e) => setClientId(e.target.value)}
        fullWidth
        margin="normal"
      />
      <TextField
        label="Client Secret"
        type="password"
        value={clientSecret}
        onChange={(e) => setClientSecret(e.target.value)}
        fullWidth
        margin="normal"
      />
      <Button variant="contained" onClick={handleSave}>
        Save Settings
      </Button>
    </Box>
  );
};

export default ApiSettings;