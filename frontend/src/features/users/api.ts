import { api } from '@/lib/api-client';
import type { ApiEnvelope } from '@/types/api';
import type {
  CreateUserRequest,
  ResetPasswordResult,
  RoleInfo,
  UpdateUserRequest,
  User,
  UserFilter,
  UserListData,
} from './types';

function pageQuery(filter: object): string {
  const params = new URLSearchParams();
  Object.entries(filter).forEach(([key, value]) => {
    if (value !== undefined && value !== null && String(value).trim() !== '') {
      params.set(key, String(value));
    }
  });
  const query = params.toString();
  return query ? `?${query}` : '';
}

export async function listUsers(filter: UserFilter = {}): Promise<ApiEnvelope<UserListData>> {
  return api.getEnvelope<UserListData>(`/users${pageQuery(filter)}`);
}

export async function getUser(id: string): Promise<User> {
  return api.get<User>(`/users/${id}`);
}

export async function createUser(payload: CreateUserRequest): Promise<User> {
  return api.post<User>('/users', payload);
}

export async function updateUser(id: string, payload: UpdateUserRequest): Promise<User> {
  return api.patch<User>(`/users/${id}`, payload);
}

export async function deleteUser(id: string): Promise<void> {
  await api.delete<{ message: string }>(`/users/${id}`);
}

export async function resetUserPassword(id: string): Promise<ResetPasswordResult> {
  return api.post<ResetPasswordResult>(`/users/${id}/reset-password`);
}

export async function replaceUserOutlets(id: string, outletIds: string[]): Promise<User> {
  return api.put<User>(`/users/${id}/outlets`, { outlet_ids: outletIds });
}

export async function listRoles(): Promise<RoleInfo[]> {
  return api.get<RoleInfo[]>('/roles');
}
