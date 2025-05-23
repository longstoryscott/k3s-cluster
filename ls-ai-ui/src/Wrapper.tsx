import { useMemo } from 'react'
import App from './App'
import { useAuth } from './auth'
import { ChatProvider } from './chat'
import { ThemeProvider } from '@emotion/react'
import useColorMode from './hooks/useColorMode'
import config from './config/index'
import { CssBaseline } from '@mui/material'
import { ConfigProvider } from './context/ConfigContext'
import LoginPage from './pages/LoginPage'
import MainLayout from './components/Layout/MainLayout'
import ThemeToggle from './components/Shared/ThemeToggle'
import LoadingAnimation from './components/Shared/LoadingAnimation'


const Wrapper:React.FC = () => {
  const auth = useAuth();
  const [mode, setMode] = useColorMode();

  const theme = useMemo(() => {
    return mode === 'dark' ? config.theme.dark : config.theme.light;
  }, [mode]);

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <ChatProvider>
        <ConfigProvider>
          {auth.evaluating ? (
            <LoadingAnimation size={600} />
          ) : auth.isAuthenticated ? (
            <MainLayout><App /></MainLayout>
          ) : (
            <LoginPage />
          )}
        </ConfigProvider>
      </ChatProvider>
      <ThemeToggle mode={mode} setMode={setMode} />
    </ThemeProvider>
  )
}

export default Wrapper;
