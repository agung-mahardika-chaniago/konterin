export function formatCurrency(value: number): string {
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    maximumFractionDigits: 0,
  }).format(value);
}

export function formatCompactCurrency(value: number): string {
  const absolute = Math.abs(value);
  if (absolute >= 1_000_000_000) return `Rp${(value / 1_000_000_000).toLocaleString('id-ID', { maximumFractionDigits: 1 })} M`;
  if (absolute >= 1_000_000) return `Rp${(value / 1_000_000).toLocaleString('id-ID', { maximumFractionDigits: 1 })} jt`;
  if (absolute >= 1_000) return `Rp${(value / 1_000).toLocaleString('id-ID', { maximumFractionDigits: 0 })} rb`;
  return `Rp${value.toLocaleString('id-ID')}`;
}

export function formatNumber(value: number): string {
  return new Intl.NumberFormat('id-ID').format(value);
}

export function formatDuration(minutes: number): string {
  if (minutes <= 0) return '—';
  const hours = Math.floor(minutes / 60);
  const rest = Math.round(minutes % 60);
  if (hours === 0) return `${rest}m`;
  if (rest === 0) return `${hours}j`;
  return `${hours}j ${rest}m`;
}

export function localDateKey(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

/** Inclusive range ending today, in the browser's local calendar. */
export function rangeFor(days: number): { from: string; to: string } {
  const to = new Date();
  const from = new Date(to.getFullYear(), to.getMonth(), to.getDate() - (days - 1));
  return { from: localDateKey(from), to: localDateKey(to) };
}

export function greetingFor(date: Date): string {
  const hour = date.getHours();
  if (hour < 11) return 'Selamat pagi';
  if (hour < 15) return 'Selamat siang';
  if (hour < 18) return 'Selamat sore';
  return 'Selamat malam';
}

export function longToday(): string {
  const label = new Intl.DateTimeFormat('id-ID', {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  }).format(new Date());
  return label.charAt(0).toUpperCase() + label.slice(1);
}

export function initials(name: string): string {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase() ?? '')
    .join('');
}
