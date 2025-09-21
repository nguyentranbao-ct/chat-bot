import { User } from '../types';

export const getStoredToken = (): string | null => {
  return localStorage.getItem('auth_token');
};

export const setStoredToken = (token: string): void => {
  localStorage.setItem('auth_token', token);
};

export const removeStoredToken = (): void => {
  localStorage.removeItem('auth_token');
};

export const getStoredUser = (): User | null => {
  const userStr = localStorage.getItem('user');
  if (!userStr) return null;

  try {
    return JSON.parse(userStr) as User;
  } catch (error) {
    console.error('Failed to parse stored user:', error);
    return null;
  }
};

export const setStoredUser = (user: User): void => {
  localStorage.setItem('user', JSON.stringify(user));
};

export const removeStoredUser = (): void => {
  localStorage.removeItem('user');
};

export const isAuthenticated = (): boolean => {
  const token = getStoredToken();
  const user = getStoredUser();
  return !!(token && user);
};

export const clearAuth = (): void => {
  removeStoredToken();
  removeStoredUser();
};