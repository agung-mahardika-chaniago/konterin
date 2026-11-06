const MOVEMENT_LABELS: Record<string, string> = {
  initial: 'Stok Awal',
  purchase_in: 'Pembelian',
  service_out: 'Pemakaian Servis',
  service_return: 'Pengembalian Servis',
  adjustment_in: 'Penyesuaian Masuk',
  adjustment_out: 'Penyesuaian Keluar',
  transfer_in: 'Transfer Masuk',
  transfer_out: 'Transfer Keluar',
};

const MOVEMENT_VARIANTS: Record<string, 'neutral' | 'brand' | 'success' | 'warning' | 'danger' | 'info' | 'violet'> = {
  initial: 'neutral',
  purchase_in: 'success',
  service_out: 'info',
  service_return: 'warning',
  adjustment_in: 'violet',
  adjustment_out: 'danger',
  transfer_in: 'brand',
  transfer_out: 'warning',
};

export function movementTypeLabel(type: string): string {
  return MOVEMENT_LABELS[type] ?? type;
}

export function movementTypeVariant(type: string): 'neutral' | 'brand' | 'success' | 'warning' | 'danger' | 'info' | 'violet' {
  return MOVEMENT_VARIANTS[type] ?? 'neutral';
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

export function formatDate(value?: string | null): string {
  if (!value) return '—';
  return new Date(value).toLocaleDateString('id-ID', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  });
}

export function todayInput(): string {
  const now = new Date();
  const offset = now.getTimezoneOffset();
  return new Date(now.getTime() - offset * 60_000).toISOString().slice(0, 10);
}
