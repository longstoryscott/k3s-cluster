import { useState } from 'react';
import { Button, TextField, Grid, Typography } from '@mui/material';
import { useAuth } from '../auth';
import { useNavigate } from 'react-router-dom';

const LoginPage = () => {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const auth = useAuth();
  const navigate = useNavigate();

  const handleLogin = async () => {
    try {
      await auth.login(username, password);
      navigate('/'); // Redirect to chat page after successful login
    } catch (error) {
      console.error('Login failed:', error);
      // Handle login error (e.g., show a notification)
    }
  };

  return (
    <Grid container spacing={2} style={{ padding: 20 }}>
      <Grid size={12} component="div">
        <Typography variant="h4">Login</Typography>
      </Grid>
      <Grid size={12} component="div">
        <TextField
          label="Username"
          variant="outlined"
          fullWidth
          value={username}
          onChange={(e) => setUsername(e.target.value)}
        />
      </Grid>
      <Grid  size={12} component="div">
        <TextField
          label="Password"
          type="password"
          variant="outlined"
          fullWidth
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />
      </Grid>
      <Grid size={12} component="div">
        <Button variant="contained" color="primary" onClick={handleLogin}>
          Login
        </Button>
      </Grid>
    </Grid>
  );
};

export default LoginPage;