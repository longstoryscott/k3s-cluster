import React from 'react';
import { Switch, FormControlLabel } from '@mui/material';
import { useTheme } from '@mui/material/styles';

interface ThemeToggleProps {
  mode: 'light' | 'dark' | 'system';
  setMode: React.Dispatch<React.SetStateAction<'light' | 'dark' | 'system'>>;
}

const ThemeToggle: React.FC<ThemeToggleProps> = ({ mode, setMode }) => {
  const theme = useTheme();

  const handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setMode(event.target.checked ? 'dark' : 'light');
  };

  return (
    <FormControlLabel
      control={
        <Switch
          checked={mode === 'dark'}
          onChange={handleChange}
          name="themeToggle"
          color="primary"
        />
      }
      label={mode === 'dark' ? 'Switch to Light Mode' : 'Switch to Dark Mode'}
      style={{ color: theme.palette.text.primary }}
    />
  );
};

export default ThemeToggle;