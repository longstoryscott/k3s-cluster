import React from 'react';
import Button from '@mui/material/Button';
import config from '../config';

const LogoutButton: React.FC = () => {
  const handleLogout = () => {
    // Clear tokens from sessionStorage
    sessionStorage.removeItem('tokens');

    fetch(`${config.auth.logoutEndpoint}`, {
      method: 'GET',
      credentials: 'include'
    }).finally(() => {
      // Redirect to login or home
      window.location.href = '/';
    });
  };

  return (
    <Button
      variant="contained"
      color="secondary"
      onClick={handleLogout}
    >
      Logout
    </Button>
  );
};

export default LogoutButton;
