import {
  // useEffect,
  useState,
  ReactNode,
  useEffect
} from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { AuthContext } from './useAuth';
import { UserManager, Log, User } from 'oidc-client-ts';
import { userManager, logoutSession } from './userManager';
Log.setLogger(console);
export interface AuthContextType {
    isAuthenticated: boolean;
    evaluating: boolean;
    userManager: UserManager;
    user?: User;
    logout: () => Promise<void>;
}

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [isAuthenticated] = useState(true);
  const [evaluating, setEvaluating] = useState(true);
  const [user, setUser] = useState<User>();
  const location = useLocation();
  const navigate = useNavigate();

  const logout = async () => {
    await logoutSession();
    setUser(undefined);
    setEvaluating(true);
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
