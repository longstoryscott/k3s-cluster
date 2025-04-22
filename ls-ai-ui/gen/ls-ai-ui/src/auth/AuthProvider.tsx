import {
  createContext,
  useContext,
  useEffect,
  useState,
  ReactNode,
  useCallback
} from 'react';
import config from '../config';

interface AuthContextType {
    user: User;
    isAuthenticated: boolean;
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

const AuthContext = createContext<AuthContextType | undefined>(undefined);

let refreshTimer: ReturnType<typeof setTimeout>;

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [user, setUser] = useState<User>(defaultUser);
  const [isAuthenticated, setIsAuthenticated] = useState(false);

  const clearSession = () => {
    sessionStorage.removeItem(config.auth.tokenStorageKey);
    sessionStorage.removeItem(config.auth.userStorageKey);
    setUser(defaultUser);
    setIsAuthenticated(false);
    if (refreshTimer) {
      clearTimeout(refreshTimer);
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

  const login = useCallback(async (username: string, password: string) => {
    const formData = new URLSearchParams();
    formData.append('grant_type', 'password');
    formData.append('username', username);
    formData.append('password', password);
    formData.append('client_id', config.auth.clientId);
    formData.append('scope', 'openid profile email offline_access');

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
    setUser({...payload, accessToken: tokens.access_token, refreshToken: tokens.refresh_token});
    setIsAuthenticated(true);

    const expiresIn = tokens.expires_in || 300;
    scheduleRefresh(expiresIn, tokens.refresh_token);
  }, []);

  const logout = useCallback(() => {
    clearSession();
  }, []);

  useEffect(() => {
    const tokensStr = sessionStorage.getItem(config.auth.tokenStorageKey);
    const userStr = sessionStorage.getItem(config.auth.userStorageKey);

    if (tokensStr && userStr) {
      try {
        const tokens = JSON.parse(tokensStr);
        const storedUser = JSON.parse(userStr);

        const expiresIn = tokens.expires_in || 300;
        scheduleRefresh(expiresIn, tokens.refresh_token);

        setUser({...storedUser, accessToken: tokens.access_token, refreshToken: tokens.refresh_token});
        setIsAuthenticated(true);
      } catch {
        clearSession();
      }
    }
  }, []);

  return (
    <AuthContext.Provider value={{ user, isAuthenticated, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error('useAuth must be used within AuthProvider');
  }
  return ctx;
};