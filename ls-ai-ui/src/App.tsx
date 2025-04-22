import './App.css';
import { useMemo } from 'react';
import { CssBaseline, ThemeProvider } from '@mui/material';
import { useAuth } from './auth';
import config from './config';
import MainLayout from './components/Layout/MainLayout';
import ChatPage from './pages/ChatPage';
import LoginPage from './pages/LoginPage';
import SettingsPage from './pages/SettingsPage';
import { BrowserRouter as Router, Route, Routes } from 'react-router-dom';
import ThemeToggle from './components/Shared/ThemeToggle';
import useColorMode from './hooks/useColorMode';

function App() {
  const [mode, setMode] = useColorMode();
  const auth = useAuth();

  const theme = useMemo(() => {
    return mode === 'dark' ? config.theme.dark : config.theme.light;
  }, [mode]);

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
      <ThemeToggle mode={mode} setMode={setMode}  />
    </ThemeProvider>
  );
}

export default App;