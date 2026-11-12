import { api, apiDownload, saveBlob } from '@/lib/api-client';
import type { ApiEnvelope } from '@/types/api';
import type { Backup, Settings, SettingsForm } from './types';

export async function getSettings(): Promise<Settings> {
  return api.get<Settings>('/settings');
}

export async function updateSettings(form: SettingsForm): Promise<Settings> {
  return api.patch<Settings>('/settings', form);
}

export async function listBackups(): Promise<ApiEnvelope<Backup[]>> {
  return api.getEnvelope<Backup[]>('/settings/backups');
}

export async function createBackup(): Promise<Backup> {
  return api.post<Backup>('/settings/backups');
}

export async function downloadBackup(id: string): Promise<void> {
  const file = await apiDownload(`/settings/backups/${id}/download`);
  saveBlob(file.blob, file.fileName);
}
