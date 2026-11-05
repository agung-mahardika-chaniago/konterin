import { api } from '@/lib/api-client';

import type { DashboardAlerts, DashboardCharts, DashboardSummary, DashboardTechnician } from './types';

function query(params: Record<string, string | undefined>): string {
  const search = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value) search.set(key, value);
  });
  const encoded = search.toString();
  return encoded ? `?${encoded}` : '';
}

export function getSummary(date: string): Promise<DashboardSummary> {
  return api.get<DashboardSummary>(`/dashboard/summary${query({ date })}`);
}

export function getCharts(from: string, to: string): Promise<DashboardCharts> {
  return api.get<DashboardCharts>(`/dashboard/charts${query({ from, to })}`);
}

export function getAlerts(): Promise<DashboardAlerts> {
  return api.get<DashboardAlerts>('/dashboard/alerts');
}

export function getTechnicians(from: string, to: string): Promise<DashboardTechnician[]> {
  return api.get<DashboardTechnician[]>(`/dashboard/technicians${query({ from, to })}`);
}
