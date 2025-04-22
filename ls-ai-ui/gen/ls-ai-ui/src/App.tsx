import { useState, useMemo } from 'react';
import { CssBaseline, ThemeProvider, useMediaQuery } from '@mui/material';
import { useAuth } from './auth/AuthProvider';
import config from './config';
import MainLayout from './components/Layout/MainLayout';
import ChatPage from './pages/ChatPage';
import LoginPage from './pages/LoginPage';
import SettingsPage from './pages/SettingsPage';
import { BrowserRouter as Router, Route, Routes } from 'react-router-dom';
import { ThemeToggle } from './components/Shared/ThemeToggle';

function App() {
  const [mode, setMode] = useState<'light' | 'dark' | 'system'>('system');
  const prefersDarkMode = useMediaQuery('(prefers-color-scheme: dark)');
  const auth = useAuth();

  const theme = useMemo(() => {
    if (mode === 'system') {
      return prefersDarkMode ? config.theme.dark : config.theme.light;
    }
    return mode === 'dark' ? config.theme.dark : config.theme.light;
  }, [mode, prefersDarkMode]);

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <Router>
        <MainLayout>
          <Routes>
            <Route path="/" element={auth.isAuthenticated ? <ChatPage /> : <LoginPage />} />
            <Route path="/settings" element={<SettingsPage />} />
          </Routes>
        </MainLayout>
      </Router>
      <ThemeToggle mode={mode} setMode={setMode} />
    </ThemeProvider>
  );
}

export default App;