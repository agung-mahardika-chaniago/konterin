import type { ClaimStatus, WarrantyStatus, WarrantyType } from './types';

type BadgeVariant = 'neutral' | 'brand' | 'success' | 'warning' | 'danger' | 'info' | 'violet';

export const WARRANTY_TYPE: Record<WarrantyType, { label: string; variant: BadgeVariant }> = {
  service: { label: 'Jasa Servis', variant: 'brand' },
  part: { label: 'Sparepart', variant: 'violet' },
  both: { label: 'Jasa + Sparepart', variant: 'info' },
};

export const WARRANTY_STATUS: Record<WarrantyStatus, { label: string; variant: BadgeVariant }> = {
  active: { label: 'Aktif', variant: 'success' },
  expired: { label: 'Berakhir', variant: 'neutral' },
  void: { label: 'Dibatalkan', variant: 'danger' },
};

export const CLAIM_STATUS: Record<ClaimStatus, { label: string; variant: BadgeVariant }> = {
  submitted: { label: 'Menunggu Review', variant: 'warning' },
  approved: { label: 'Disetujui', variant: 'info' },
  in_progress: { label: 'Dikerjakan Ulang', variant: 'brand' },
  completed: { label: 'Selesai', variant: 'success' },
  rejected: { label: 'Ditolak', variant: 'danger' },
};

export const CLAIM_TRANSITIONS: Record<ClaimStatus, ClaimStatus[]> = {
  submitted: ['approved', 'in_progress', 'rejected'],
  approved: ['in_progress', 'completed', 'rejected'],
  in_progress: ['completed', 'rejected'],
  completed: [],
  rejected: [],
};

export function warrantyTypeLabel(type: string): string {
  return WARRANTY_TYPE[type as WarrantyType]?.label ?? type;
}

export function warrantyStatusLabel(status: string): string {
  return WARRANTY_STATUS[status as WarrantyStatus]?.label ?? status;
}

export function claimStatusLabel(status: string): string {
  return CLAIM_STATUS[status as ClaimStatus]?.label ?? status;
}

export function formatDate(value?: string | null): string {
  if (!value) return '—';
  return new Date(`${value}T00:00:00`).toLocaleDateString('id-ID', {
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  });
}

export function formatDateTime(value?: string | null): string {
  if (!value) return '—';
  return new Date(value).toLocaleString('id-ID', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

export function daysRemaining(endDate: string): number {
  const end = new Date(`${endDate}T23:59:59`);
  const today = new Date();
  return Math.ceil((end.getTime() - today.getTime()) / 86_400_000);
}
