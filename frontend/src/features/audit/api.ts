import { api } from '@/lib/api-client';
import type { ApiEnvelope } from '@/types/api';
import type { AuditFilter, AuditListData } from './types';

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

export async function listAuditLogs(filter: AuditFilter = {}): Promise<ApiEnvelope<AuditListData>> {
  return api.getEnvelope<AuditListData>(`/audit-logs${pageQuery(filter)}`);
}

export async function listAuditModules(): Promise<string[]> {
  return api.get<string[]>('/audit-logs/modules');
}
