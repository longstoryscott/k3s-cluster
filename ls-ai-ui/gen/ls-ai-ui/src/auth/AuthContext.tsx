import { createContext, useContext, useState, ReactNode, useEffect } from 'react';
import config from '../config';

interface AuthContextType {
    user: User | null;
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
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [user, setUser] = useState<User | null>(null);
  const [isAuthenticated, setIsAuthenticated] = useState(false);

  const login = async (username: string, password: string) => {
    const formData = new URLSearchParams();
    formData.append('grant_type', 'password');
    formData.append('username', username);
    formData.append('password', password);
    formData.append('client_id', config.auth.clientId);

    const response = await fetch(config.auth.tokenEndpoint, {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: formData.toString()
    });

    if (!response.ok) {
      throw new Error('Login failed');
    }

    const tokens = await response.json();
    const payload = JSON.parse(atob(tokens.id_token.split('.')[1]));
    setUser({ ...payload, accessToken: tokens.access_token, refreshToken: tokens.refresh_token });
    setIsAuthenticated(true);
  };

  const logout = () => {
    setUser(null);
    setIsAuthenticated(false);
    sessionStorage.clear();
  };

  useEffect(() => {
    const tokensStr = sessionStorage.getItem(config.auth.tokenStorageKey);
    const userStr = sessionStorage.getItem(config.auth.userStorageKey);

    if (tokensStr && userStr) {
      const tokens = JSON.parse(tokensStr);
      const storedUser = JSON.parse(userStr);
      setUser({ ...storedUser, accessToken: tokens.access_token, refreshToken: tokens.refresh_token });
      setIsAuthenticated(true);
    }
  }, []);

  return (
    <AuthContext.Provider value={{ user, isAuthenticated, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};