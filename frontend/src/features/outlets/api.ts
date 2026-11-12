import { api } from '@/lib/api-client';
import type { CreateOutletRequest, Outlet, UpdateOutletRequest } from './types';

export async function listOutlets(): Promise<Outlet[]> {
  return api.get<Outlet[]>('/outlets');
}

export async function createOutlet(payload: CreateOutletRequest): Promise<Outlet> {
  return api.post<Outlet>('/outlets', payload);
}

export async function updateOutlet(id: string, payload: UpdateOutletRequest): Promise<Outlet> {
  return api.patch<Outlet>(`/outlets/${id}`, payload);
}
