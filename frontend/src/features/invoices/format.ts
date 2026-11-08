import type { InvoiceStatus, PaymentMethod, PaymentStatus } from './types';

export const INVOICE_STATUS: Record<
  InvoiceStatus,
  { label: string; variant: 'neutral' | 'brand' | 'success' | 'warning' | 'danger' | 'info' | 'violet' }
> = {
  draft: { label: 'Draft', variant: 'neutral' },
  unpaid: { label: 'Belum Dibayar', variant: 'warning' },
  partial: { label: 'Dibayar Sebagian', variant: 'info' },
  paid: { label: 'Lunas', variant: 'success' },
  void: { label: 'Dibatalkan', variant: 'danger' },
};

export const PAYMENT_METHOD: Record<PaymentMethod, { label: string; short: string }> = {
  cash: { label: 'Tunai', short: 'Tunai' },
  bank_transfer: { label: 'Transfer Bank', short: 'Transfer' },
  qris: { label: 'QRIS', short: 'QRIS' },
  card: { label: 'Kartu Debit/Kredit', short: 'Kartu' },
  other: { label: 'Lainnya', short: 'Lainnya' },
};

export const PAYMENT_STATUS: Record<PaymentStatus, { label: string; variant: 'success' | 'danger' }> = {
  posted: { label: 'Berhasil', variant: 'success' },
  void: { label: 'Dibatalkan', variant: 'danger' },
};

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
    month: 'long',
    year: 'numeric',
  });
}

export function invoiceStatusLabel(status: string): string {
  return INVOICE_STATUS[status as InvoiceStatus]?.label ?? status;
}

export function paymentMethodLabel(method: string): string {
  return PAYMENT_METHOD[method as PaymentMethod]?.label ?? method;
}

export function itemTypeLabel(itemType: string): string {
  switch (itemType) {
    case 'service':
      return 'Jasa';
    case 'sparepart':
      return 'Sparepart';
    case 'custom':
      return 'Custom';
    default:
      return itemType;
  }
}

export function todayInput(): string {
  const now = new Date();
  const year = now.getFullYear();
  const month = String(now.getMonth() + 1).padStart(2, '0');
  const day = String(now.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}
