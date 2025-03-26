/**
 * useAuth.tsx
 * 
 * @module hooks
 * @description Custom hook for authentication and user management
 * @version 1.0.0
 */

import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import axios from 'axios';
import { useConfig } from './useConfig';

// User interface
export interface User {
  id: string;
  username: string;
  email: string;
  role: string;
  isAdmin: boolean;
  mfaEnabled: boolean;
  lastLogin?: string;
  createdAt: string;
}

// Admin creation payload
export interface AdminCreationPayload {
  username: string;
  password: string;
  email: string;
  mfaEnabled: boolean;
  rememberDevice: boolean;
}

// Authentication context interface
interface AuthContextType {
  user: User | null;
  loading: boolean;
  error: string | null;
  isAuthenticated: boolean;
  login: (username: string, password: string, mfaCode?: string) => Promise<boolean>;
  logout: () => Promise<void>;
  createAdmin: (payload: AdminCreationPayload) => Promise<boolean>;
  resetPassword: (email: string) => Promise<boolean>;
  updateUser: (updates: Partial<User>) => Promise<boolean>;
}

// Create the auth context
const AuthContext = createContext<AuthContextType>({
  user: null,
  loading: false,
  error: null,
  isAuthenticated: false,
  login: async () => false,
  logout: async () => {},
  createAdmin: async () => false,
  resetPassword: async () => false,
  updateUser: async () => false,
});

interface AuthProviderProps {
  children: ReactNode;
}

export const AuthProvider: React.FC<AuthProviderProps> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const { config } = useConfig();

  // Determine if authentication is required
  const authRequired = config?.security?.authentication?.enabled ?? false;

  // Check for existing session on mount
  useEffect(() => {
    const checkAuth = async () => {
      setLoading(true);
      
      try {
        if (!authRequired) {
          setUser(null);
          return;
        }
        
        const token = localStorage.getItem('auth_token');
        if (!token) {
          setUser(null);
          return;
        }
        
        // Set default auth header for all requests
        axios.defaults.headers.common['Authorization'] = `Bearer ${token}`;
        
        // Check if the token is valid
        const response = await axios.get('/api/v1/auth/me');
        setUser(response.data);
      } catch (err) {
        console.error('Auth check failed:', err);
        setUser(null);
        localStorage.removeItem('auth_token');
        delete axios.defaults.headers.common['Authorization'];
      } finally {
        setLoading(false);
      }
    };
    
    checkAuth();
  }, [authRequired]);

  // Login function
  const login = async (username: string, password: string, mfaCode?: string): Promise<boolean> => {
    setLoading(true);
    setError(null);
    
    try {
      const response = await axios.post('/api/v1/auth/login', {
        username,
        password,
        mfaCode,
      });
      
      const { token, user: userData } = response.data;
      
      // Store token and set user
      localStorage.setItem('auth_token', token);
      axios.defaults.headers.common['Authorization'] = `Bearer ${token}`;
      setUser(userData);
      
      return true;
    } catch (err: any) {
      const errorMessage = err.response?.data?.message || 'Login failed';
      setError(errorMessage);
      return false;
    } finally {
      setLoading(false);
    }
  };

  // Logout function
  const logout = async (): Promise<void> => {
    try {
      await axios.post('/api/v1/auth/logout');
    } catch (err) {
      console.error('Logout API call failed:', err);
    } finally {
      // Always clear local state regardless of API success
      localStorage.removeItem('auth_token');
      delete axios.defaults.headers.common['Authorization'];
      setUser(null);
    }
  };

  // Create admin function
  const createAdmin = async (payload: AdminCreationPayload): Promise<boolean> => {
    setLoading(true);
    setError(null);
    
    try {
      const response = await axios.post('/api/v1/auth/create-admin', payload);
      
      // If we get here, admin creation was successful
      // For the onboarding flow, we may want to auto-login
      if (response.data.autoLogin) {
        const { token, user: userData } = response.data;
        localStorage.setItem('auth_token', token);
        axios.defaults.headers.common['Authorization'] = `Bearer ${token}`;
        setUser(userData);
      }
      
      return true;
    } catch (err: any) {
      const errorMessage = err.response?.data?.message || 'Failed to create admin account';
      setError(errorMessage);
      return false;
    } finally {
      setLoading(false);
    }
  };

  // Reset password function
  const resetPassword = async (email: string): Promise<boolean> => {
    setLoading(true);
    setError(null);
    
    try {
      await axios.post('/api/v1/auth/reset-password', { email });
      return true;
    } catch (err: any) {
      const errorMessage = err.response?.data?.message || 'Password reset failed';
      setError(errorMessage);
      return false;
    } finally {
      setLoading(false);
    }
  };

  // Update user function
  const updateUser = async (updates: Partial<User>): Promise<boolean> => {
    if (!user) return false;
    
    setLoading(true);
    setError(null);
    
    try {
      const response = await axios.patch('/api/v1/auth/user', updates);
      setUser({ ...user, ...response.data });
      return true;
    } catch (err: any) {
      const errorMessage = err.response?.data?.message || 'Failed to update user';
      setError(errorMessage);
      return false;
    } finally {
      setLoading(false);
    }
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        loading,
        error,
        isAuthenticated: !!user,
        login,
        logout,
        createAdmin,
        resetPassword,
        updateUser,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
};

// Hook for using the auth context
export const useAuth = () => useContext(AuthContext);
