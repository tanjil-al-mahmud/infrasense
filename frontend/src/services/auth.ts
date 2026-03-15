import apiClient from './api';

/**
 * Authentication Service
 * 
 * Handles user authentication operations:
 * - Login (POST /api/v1/auth/login)
 * - Logout (POST /api/v1/auth/logout)
 * - Get current user (GET /api/v1/auth/me)
 * - JWT token storage in localStorage
 */

const TOKEN_KEY = 'infrasense_token';

export interface LoginCredentials {
  username: string;
  password: string;
}

export interface LoginResponse {
  token: string;
  user: User;
}

export interface User {
  id: string;
  username: string;
  email?: string;
  full_name?: string;
  role: 'admin' | 'operator' | 'viewer';
  enabled: boolean;
  last_login_at?: string;
}

/**
 * Login user with username and password
 * Stores JWT token in localStorage on success
 * 
 * @param credentials - Username and password
 * @returns Login response with token and user info
 * @throws Error if login fails
 */
export const login = async (credentials: LoginCredentials): Promise<LoginResponse> => {
  const response = await apiClient.post<LoginResponse>('/auth/login', credentials);
  
  // Store JWT token in localStorage
  if (response.data.token) {
    localStorage.setItem(TOKEN_KEY, response.data.token);
  }
  
  return response.data;
};

/**
 * Logout current user
 * Clears JWT token from localStorage
 * 
 * @throws Error if logout request fails
 */
export const logout = async (): Promise<void> => {
  try {
    await apiClient.post('/auth/logout');
  } finally {
    // Always clear token from localStorage, even if API call fails
    localStorage.removeItem(TOKEN_KEY);
  }
};

/**
 * Get current authenticated user information
 * 
 * @returns Current user info
 * @throws Error if not authenticated or request fails
 */
export const getCurrentUser = async (): Promise<User> => {
  const response = await apiClient.get<User>('/auth/me');
  return response.data;
};

/**
 * Get stored JWT token from localStorage
 * 
 * @returns JWT token or null if not found
 */
export const getToken = (): string | null => {
  return localStorage.getItem(TOKEN_KEY);
};

/**
 * Check if user is authenticated (has valid token)
 * Note: This only checks if token exists, not if it's valid
 * 
 * @returns true if token exists in localStorage
 */
export const isAuthenticated = (): boolean => {
  return getToken() !== null;
};
