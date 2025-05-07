import { AppBar, Toolbar, Typography, Button, useTheme } from '@mui/material';
import { useAuth } from '../../auth';

const TopBar = () => {
  const { user, logout } = useAuth();
  const theme = useTheme();

  return (
    <AppBar position="static">
      <Toolbar>
        <Typography variant="h6" sx={{ flexGrow: 1 }}>
          Chat Application
        </Typography>
        {user.name && (
          <Typography variant="body1" sx={{ mr: theme.spacing(2.5) }}>
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