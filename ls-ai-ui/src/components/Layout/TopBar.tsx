import { AppBar, Toolbar, Typography, Button } from '@mui/material';
import { useAuth } from '../../auth';

const TopBar = () => {
  const { user, logout } = useAuth();

  return (
    <AppBar position="static">
      <Toolbar>
        <Typography variant="h6" style={{ flexGrow: 1 }}>
          Chat Application
        </Typography>
        {user.name && (
          <Typography variant="body1" style={{ marginRight: '20px' }}>
            Welcome, {user.name}
          </Typography>
        )}
        <Button color="inherit" onClick={logout}>
          Logout
        </Button>
      </Toolbar>
    </AppBar>
  );
};

export default TopBar;