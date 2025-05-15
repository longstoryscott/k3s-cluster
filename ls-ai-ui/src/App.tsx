import './App.css';
import { useMemo } from 'react';
import { CssBaseline, LinearProgress, ThemeProvider } from '@mui/material';
import { useAuth } from './auth';
import config from './config';
import MainLayout from './components/Layout/MainLayout';
import ChatPage from './pages/ChatPage';
import LoginPage from './pages/LoginPage';
import SettingsPage from './pages/SettingsPage';
import { Route, Routes, Navigate, useParams } from 'react-router-dom';
import ThemeToggle from './components/Shared/ThemeToggle';
import useColorMode from './hooks/useColorMode';
import { ConfigProvider } from './context/ConfigContext';
import ModelProfilesPage from './pages/ModelProfilesPage';

function App() {
  const [mode, setMode] = useColorMode();
  const auth = useAuth();
  const params = useParams();

  const theme = useMemo(() => {
    return mode === 'dark' ? config.theme.dark : config.theme.light;
  }, [mode]);

  

  return (auth.evaluating ? (
    <LinearProgress />
  ) : (
    <ConfigProvider>
      <ThemeProvider theme={theme}>
        <CssBaseline />
        <Routes>
          {/* Login route outside of MainLayout */}
          <Route path="/login" element={
            auth.isAuthenticated ? <Navigate to="/" /> : <LoginPage />
          } />
          
          {/* Protected routes inside MainLayout */}
          <Route path="/" element={
            auth.isAuthenticated ? <MainLayout> <ChatPage /> </MainLayout> : <Navigate to="/login" />
          } />
          <Route path="/chat/:conversationId" element={
            auth.isAuthenticated ? <MainLayout> <ChatPage /> </MainLayout> : <Navigate to={`/login?redirect=/chat/${params.conversationId}`} />
          } />
          <Route path="/settings" element={
            auth.isAuthenticated ? <MainLayout> <SettingsPage /> </MainLayout> : <Navigate to="/login?redirect=/settings" />
          } />
          <Route path="/model-profiles" element={
            auth.isAuthenticated ? <MainLayout> <ModelProfilesPage /> </MainLayout> : <Navigate to="/login?redirect=/model-profiles" />
          } />
        </Routes>
        <ThemeToggle mode={mode} setMode={setMode} />
      </ThemeProvider>
    </ConfigProvider>
  ));
}

export default App;