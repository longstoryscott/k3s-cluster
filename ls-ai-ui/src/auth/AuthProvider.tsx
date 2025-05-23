import {
  useEffect,
  useState,
  ReactNode
} from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import config from '../config';
import { AuthContext } from './useAuth';

export interface AuthContextType {
    user: User;
    isAuthenticated: boolean;
    evaluating: boolean;
    login: (username: string, password: string) => Promise<void>;
    logout: () => void;
}

interface User {
    sub: string;
    name: string;
    email: string;
    accessToken?: string;
    refreshToken?: string;
    idToken?: string;
    [key: string]: string | number | boolean | undefined;
}

const defaultUser: User = {
  sub: '',
  name: '',
  email: ''
};


let refreshTimer: ReturnType<typeof setTimeout>;

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [user, setUser] = useState<User>(defaultUser);
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [evaluating, setEvaluating] = useState(true);
  const navigate = useNavigate();
  const location = useLocation();

  const clearSession = () => {
    sessionStorage.removeItem(config.auth.tokenStorageKey);
    sessionStorage.removeItem(config.auth.userStorageKey);
    setUser(defaultUser);
    setIsAuthenticated(false);
    if (refreshTimer) {
      clearTimeout(refreshTimer);
    }
    
    // Navigate to login page if not already there
    if (location.pathname !== '/login') {
      navigate('/login');
    }
  };

  const scheduleRefresh = (expiresIn: number, refreshToken: string) => {
    if (refreshTimer) {
      clearTimeout(refreshTimer);
    }

    const refreshIn = (expiresIn - 60) * 1000; // refresh 1 minute before expiration
    refreshTimer = setTimeout(() => {
      refreshTokenFlow(refreshToken);
    }, refreshIn);
  };

  const refreshTokenFlow = async (refreshToken: string) => {
    try {
      const formData = new URLSearchParams();
      formData.append('grant_type', 'refresh_token');
      formData.append('refresh_token', refreshToken);
      formData.append('client_id', config.auth.clientId);
      // formData.append('client_secret', config.auth.clientSecret);
      formData.append('scope', config.auth.scope);

      const response = await fetch(config.auth.tokenEndpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: formData.toString()
      });

      if (!response.ok) {
        throw new Error('Refresh failed');
      }

      const tokens = await response.json();
      sessionStorage.setItem(config.auth.tokenStorageKey, JSON.stringify(tokens));

      const payload = JSON.parse(atob(tokens.id_token.split('.')[1]));
      sessionStorage.setItem(config.auth.userStorageKey, JSON.stringify(payload));
      setUser({...payload, accessToken: tokens.access_token, refreshToken: tokens.refresh_token});
      setIsAuthenticated(true);

      const expiresIn = tokens.expires_in || 300;
      scheduleRefresh(expiresIn, tokens.refresh_token);
    } catch (_err) {
      console.warn('Refresh failed, forcing logout');
      clearSession();
    }
  };

  const login = async (username: string, password: string) => {
    setEvaluating(true);
    const formData = new URLSearchParams();
    formData.append('grant_type', 'password');
    formData.append('username', username);
    formData.append('password', password);
    formData.append('client_id', config.auth.clientId);
    // formData.append('client_secret', config.auth.clientSecret);
    formData.append('scope', config.auth.scope);

    const response = await fetch(config.auth.tokenEndpoint, {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: formData.toString()
    });

    if (!response.ok) {
      throw new Error('Login failed');
    }

    const tokens = await response.json();
    sessionStorage.setItem(config.auth.tokenStorageKey, JSON.stringify(tokens));

    const payload = JSON.parse(atob(tokens.id_token.split('.')[1]));
    sessionStorage.setItem(config.auth.userStorageKey, JSON.stringify(payload));
    setUser(u => ({...u, ...payload, accessToken: tokens.access_token, refreshToken: tokens.refresh_token}));
    setIsAuthenticated(true);

    const expiresIn = tokens.expires_in || 300;
    scheduleRefresh(expiresIn, tokens.refresh_token);
    setEvaluating(false);
  };

  const logout = () => {
    clearSession();
  };

  useEffect(() => {
    setEvaluating(true);
    const tokensStr = sessionStorage.getItem(config.auth.tokenStorageKey);
    const userStr = sessionStorage.getItem(config.auth.userStorageKey);

    if (tokensStr && userStr) {
      try {
        const tokens = JSON.parse(tokensStr);
        const storedUser = JSON.parse(userStr);

        const expiresIn = tokens.expires_in || 300;
        scheduleRefresh(expiresIn, tokens.refresh_token);
        setUser(u => ({...u, ...storedUser, accessToken: tokens.access_token, refreshToken: tokens.refresh_token}));

        setIsAuthenticated(true);
      } catch {
        clearSession();
      }
    } else if (location.pathname !== '/login') {
      // If not authenticated and not on login page, redirect to login
      navigate('/login');
    }

    setEvaluating(false);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [location.pathname]);

  return (
    <AuthContext.Provider value={{ user, isAuthenticated, login, logout, evaluating }}>
      {children}
    </AuthContext.Provider>
  );
};
