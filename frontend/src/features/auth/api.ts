import { api } from '@/lib/api-client';
import type { AuthSession, AuthUser, ProfileResponse } from '@/types/auth';

export async function login(email: string, password: string): Promise<AuthSession> {
  return api.post<AuthSession>('/auth/login', { email, password });
}

export async function logout(): Promise<void> {
  await api.post<{ message: string }>('/auth/logout');
}

export async function me(): Promise<AuthUser> {
  const result = await api.get<ProfileResponse>('/auth/me');
  return result.user;
}

export async function changePassword(currentPassword: string, newPassword: string): Promise<void> {
  await api.post<{ message: string }>('/auth/change-password', {
    current_password: currentPassword,
    new_password: newPassword,
  });
}

export async function updateProfile(input: {
  name: string;
  phone?: string | null;
  avatar_path?: string | null;
}): Promise<AuthUser> {
  const result = await api.patch<ProfileResponse>('/auth/profile', input);
  return result.user;
}
