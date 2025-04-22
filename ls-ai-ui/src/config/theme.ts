// theme.ts
import { createTheme, ThemeOptions } from '@mui/material/styles';

const commonOptions: ThemeOptions = {
  palette: {
    primary: {
      main: '#4caf50' // green
    },
    secondary: {
      main: '#81c784' // lighter green
    },
    contrastThreshold: 4.5
  },
  typography: {
    fontFamily: 'Roboto, sans-serif',
    button: {
      textTransform: 'none'
    }
  }
};

export const lightTheme = createTheme({
  ...commonOptions,
  palette: {
    ...commonOptions.palette,
    mode: 'light',
    background: {
      default: '#f5f5f5',
      paper: '#fff'
    },
    text: {
      primary: '#1b1b1b'
    }
  }
});

export const darkTheme = createTheme({
  ...commonOptions,
  palette: {
    ...commonOptions.palette,
    mode: 'dark',
    background: {
      default: '#121212',
      paper: '#1d1d1d'
    },
    text: {
      primary: '#ffffff'
    }
  }
});
