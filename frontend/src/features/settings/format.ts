export function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB'];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit += 1;
  }
  return `${value.toLocaleString('id-ID', { maximumFractionDigits: 1 })} ${units[unit]}`;
}

export function formatDateTime(value?: string | null): string {
  if (!value) return '—';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString('id-ID', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

export const WA_PROVIDER_LABEL: Record<string, string> = {
  noop: 'Nonaktif (tidak terkirim)',
  fonnte: 'Fonnte',
  wablas: 'Wablas',
};

export function waProviderLabel(provider: string): string {
  return WA_PROVIDER_LABEL[provider] ?? provider;
}
