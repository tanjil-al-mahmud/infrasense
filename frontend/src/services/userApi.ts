import apiClient from './api';

export interface UserRecord {
  id: string;
  username: string;
  email?: string;
  full_name?: string;
  role: 'admin' | 'operator' | 'viewer';
  enabled: boolean;
  last_login_at?: string;
}

export interface CreateUserData {
  username: string;
  email?: string;
  full_name?: string;
  role: 'admin' | 'operator' | 'viewer';
  password: string;
  enabled: boolean;
}

export interface UpdateUserData {
  email?: string;
  full_name?: string;
  role?: 'admin' | 'operator' | 'viewer';
  enabled?: boolean;
}

export interface ChangePasswordData {
  new_password: string;
}

export interface ChangeOwnPasswordData {
  current_password: string;
  new_password: string;
}

export const listUsers = async (): Promise<UserRecord[]> => {
  const response = await apiClient.get<UserRecord[]>('/users');
  return response.data;
};

export const createUser = async (data: CreateUserData): Promise<UserRecord> => {
  const response = await apiClient.post<UserRecord>('/users', data);
  return response.data;
};

export const updateUser = async (id: string, data: UpdateUserData): Promise<UserRecord> => {
  const response = await apiClient.put<UserRecord>(`/users/${id}`, data);
  return response.data;
};

export const deleteUser = async (id: string): Promise<void> => {
  await apiClient.delete(`/users/${id}`);
};

export const changePassword = async (id: string, data: ChangePasswordData): Promise<void> => {
  await apiClient.put(`/users/${id}/password`, data);
};

export const changeOwnPassword = async (data: ChangeOwnPasswordData): Promise<void> => {
  await apiClient.put('/users/me/password', data);
};

export const getMe = async (): Promise<UserRecord> => {
  const response = await apiClient.get<UserRecord>('/users/me');
  return response.data;
};

export const updateMe = async (data: UpdateUserData): Promise<UserRecord> => {
  const response = await apiClient.put<UserRecord>('/users/me', data);
  return response.data;
};
