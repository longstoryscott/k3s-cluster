import { AppBar, Toolbar, Typography, Button, useTheme, IconButton } from '@mui/material';
import MenuIcon from '@mui/icons-material/Menu';
import { useAuth } from '../../auth';

const TopBar = ({ onMenuClick }: { onMenuClick?: () => void }) => {
  const { user, logout } = useAuth();
  const theme = useTheme();

  return (
    <AppBar position="sticky">
      <Toolbar>
        {onMenuClick && (
          <IconButton
            color="inherit"
            edge="start"
            onClick={onMenuClick}
            sx={{ mr: 2 }}
          >
            <MenuIcon />
          </IconButton>
        )}
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