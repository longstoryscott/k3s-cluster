import {
  // useEffect,
  useState,
  ReactNode,
  useEffect
} from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import config from '../config';
import { AuthContext } from './useAuth';
import { UserManager, Log, User } from 'oidc-client-ts';
Log.setLogger(console);
export interface AuthContextType {
    isAuthenticated: boolean;
    evaluating: boolean;
    userManager: UserManager;
    user?: User;
    logout: () => Promise<void>;
}

const userManager = new UserManager(config.auth.oidc);

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [isAuthenticated] = useState(true);
  const [evaluating, setEvaluating] = useState(true);
  const [user, setUser] = useState<User>();
  const location = useLocation();
  const navigate = useNavigate();

  const logout = async () => {
    try {
      await userManager.removeUser()
      setUser(undefined);
      setEvaluating(true);
      window.location.href = config.auth.oidc.post_logout_redirect_uri;
    } catch (error) {
      console.error('Error during logout:', error);
    }
  };

  useEffect(() => {
    setEvaluating(true);
    (async () => {
      try {
        let usr = await userManager.getUser();
        if (!usr) {
          if (location.pathname === '/callback') {
            usr = await userManager.signinRedirectCallback();
          } else {
            await userManager.signinRedirect({ state: { some: "data" } });
          }
        }
        if (usr){
          setUser(usr);
          setEvaluating(false);
        }
      } catch (error) {
        console.error('Error getting user:', error);
      } 
    })();
  }, [location.pathname, navigate]);

  return (
    <AuthContext.Provider value={{ user, isAuthenticated, evaluating, userManager, logout }}>
      {children}
    </AuthContext.Provider>
  );
};
