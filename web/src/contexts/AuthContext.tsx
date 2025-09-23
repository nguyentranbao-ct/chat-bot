import React, { createContext, useContext, useCallback, useEffect } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { clearAuth } from '../utils/auth';
import { api } from '../utils/api';

interface AuthContextType {
  handleAuthError: () => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const navigate = useNavigate();
  const location = useLocation();

  const handleAuthError = useCallback(() => {
    // Clear auth data
    clearAuth();

    // Create return URL from current location (exclude login page)
    const returnTo = location.pathname !== '/login'
      ? encodeURIComponent(location.pathname + location.search)
      : undefined;

    // Navigate to login with return URL as query param
    const loginPath = returnTo ? `/login?returnTo=${returnTo}` : '/login';
    navigate(loginPath, { replace: true });
  }, [navigate, location]);

  // Set up auth error handler for API client
  useEffect(() => {
    api.setAuthErrorHandler(handleAuthError);
  }, [handleAuthError]);

  return (
    <AuthContext.Provider value={{ handleAuthError }}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};