import { describe, expect, it } from 'vitest';

import {
  formatCompactCurrency,
  formatCurrency,
  formatDuration,
  formatNumber,
  greetingFor,
  initials,
  localDateKey,
  rangeFor,
} from './format';

describe('dashboard format helpers', () => {
  it('formats rupiah without decimals', () => {
    const formatted = formatCurrency(1234567);
    expect(formatted).toContain('Rp');
    expect(formatted).toContain('1.234.567');
    expect(formatted).not.toContain(',00');
  });

  it('uses compact units for large amounts', () => {
    expect(formatCompactCurrency(950)).toBe('Rp950');
    expect(formatCompactCurrency(2_500)).toBe('Rp3 rb');
    expect(formatCompactCurrency(2_400_000)).toBe('Rp2,4 jt');
    expect(formatCompactCurrency(1_500_000_000)).toBe('Rp1,5 M');
  });

  it('formats plain numbers with thousand separators', () => {
    expect(formatNumber(1234567)).toBe('1.234.567');
  });

  it('formats durations from minutes', () => {
    expect(formatDuration(0)).toBe('—');
    expect(formatDuration(-5)).toBe('—');
    expect(formatDuration(45)).toBe('45m');
    expect(formatDuration(120)).toBe('2j');
    expect(formatDuration(135)).toBe('2j 15m');
  });

  it('selects a greeting by local hour', () => {
    const at = (hour: number) => new Date(2026, 10, 8, hour, 0, 0);
    expect(greetingFor(at(7))).toBe('Selamat pagi');
    expect(greetingFor(at(11))).toBe('Selamat siang');
    expect(greetingFor(at(15))).toBe('Selamat sore');
    expect(greetingFor(at(20))).toBe('Selamat malam');
  });

  it('builds initials from the first two words', () => {
    expect(initials('Budi Santoso')).toBe('BS');
    expect(initials('  ayu   lestari  ')).toBe('AL');
    expect(initials('Madonna')).toBe('M');
    expect(initials('')).toBe('');
  });

  it('keys local dates without a UTC shift', () => {
    expect(localDateKey(new Date(2026, 0, 5))).toBe('2026-01-05');
    expect(localDateKey(new Date(2026, 10, 12))).toBe('2026-11-12');
  });

  it('builds inclusive ranges ending today', () => {
    const range = rangeFor(7);
    const from = new Date(`${range.from}T00:00:00`);
    const to = new Date(`${range.to}T00:00:00`);
    const days = Math.round((to.getTime() - from.getTime()) / 86_400_000) + 1;
    expect(days).toBe(7);
    expect(range.to).toBe(localDateKey(new Date()));
  });
});
