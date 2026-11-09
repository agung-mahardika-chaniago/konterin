import type { WhatsAppStatus } from './types';

export const WHATSAPP_STATUS: Record<WhatsAppStatus, { label: string; variant: 'neutral' | 'warning' | 'info' | 'success' | 'danger' }> = {
  queued: { label: 'Menunggu', variant: 'warning' },
  processing: { label: 'Diproses', variant: 'info' },
  sent: { label: 'Terkirim', variant: 'success' },
  failed: { label: 'Gagal', variant: 'danger' },
  cancelled: { label: 'Dibatalkan', variant: 'neutral' },
};

export const TEMPLATE_LABEL: Record<string, string> = {
  status_update: 'Update Status',
  ready_pickup: 'Siap Diambil',
  invoice: 'Invoice',
  reminder: 'Pengingat',
  warranty: 'Garansi',
  claim_received: 'Klaim Diterima',
};

export function templateLabel(key: string): string {
  return TEMPLATE_LABEL[key] ?? key;
}

export function whatsappStatusLabel(status: string): string {
  return WHATSAPP_STATUS[status as WhatsAppStatus]?.label ?? status;
}

export function whatsappStatusVariant(
  status: string,
): 'neutral' | 'warning' | 'info' | 'success' | 'danger' {
  return WHATSAPP_STATUS[status as WhatsAppStatus]?.variant ?? 'neutral';
}

export function timeAgo(value: string): string {
  const date = new Date(value);
  const diff = Date.now() - date.getTime();
  const minutes = Math.floor(diff / 60_000);
  if (minutes < 1) return 'Baru saja';
  if (minutes < 60) return `${minutes} menit lalu`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours} jam lalu`;
  const days = Math.floor(hours / 24);
  if (days < 7) return `${days} hari lalu`;
  return date.toLocaleDateString('id-ID', { day: 'numeric', month: 'short', year: 'numeric' });
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
