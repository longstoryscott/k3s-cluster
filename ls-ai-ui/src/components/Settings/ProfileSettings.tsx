import { useState } from 'react';
import { Box, TextField, Button } from '@mui/material';
import { useAuth } from '../../auth';

const ProfileSettings = () => {
  const { user } = useAuth();
  const [name, setName] = useState(user?.name || '');
  const [email, setEmail] = useState(user?.email || '');

  const handleSave = () => {
    // Implement the API call to update user profile
    // When implemented, this would call a function to update the user profile
  };

  return (
    <Box component="form" sx={{ mt: 2 }}>
      <TextField
        label="Name"
        variant="outlined"
        fullWidth
        margin="normal"
        value={name}
        onChange={(e) => setName(e.target.value)}
      />
      <TextField
        label="Email"
        variant="outlined"
        fullWidth
        margin="normal"
        value={email}
        onChange={(e) => setEmail(e.target.value)}
      />
      <Button 
        variant="contained" 
        color="primary" 
        sx={{ mt: 2 }} 
        onClick={handleSave}
      >
        Save Changes
      </Button>
    </Box>
  );
};

export default ProfileSettings;