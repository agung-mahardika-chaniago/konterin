import { describe, expect, it } from 'vitest';

import {
  formatDate,
  formatDateTime,
  invoiceStatusLabel,
  itemTypeLabel,
  paymentMethodLabel,
  todayInput,
} from './format';

describe('invoice format helpers', () => {
  it('renders a placeholder for missing dates', () => {
    expect(formatDate(null)).toBe('—');
    expect(formatDate(undefined)).toBe('—');
    expect(formatDateTime(null)).toBe('—');
  });

  it('formats ISO timestamps for the id-ID locale', () => {
    const value = '2026-11-08T10:00:00+07:00';
    expect(formatDate(value)).toContain('2026');
    expect(formatDate(value)).toContain('November');
    expect(formatDateTime(value)).toMatch(/\d{2}[.:]\d{2}/);
  });

  it('maps invoice statuses to Indonesian labels and falls back to the raw value', () => {
    expect(invoiceStatusLabel('paid')).toBe('Lunas');
    expect(invoiceStatusLabel('partial')).toBe('Dibayar Sebagian');
    expect(invoiceStatusLabel('unknown')).toBe('unknown');
  });

  it('maps payment methods to Indonesian labels and falls back to the raw value', () => {
    expect(paymentMethodLabel('cash')).toBe('Tunai');
    expect(paymentMethodLabel('qris')).toBe('QRIS');
    expect(paymentMethodLabel('crypto')).toBe('crypto');
  });

  it('maps invoice item types', () => {
    expect(itemTypeLabel('service')).toBe('Jasa');
    expect(itemTypeLabel('sparepart')).toBe('Sparepart');
    expect(itemTypeLabel('custom')).toBe('Custom');
    expect(itemTypeLabel('other')).toBe('other');
  });

  it('returns an ISO date for the date input', () => {
    expect(todayInput()).toMatch(/^\d{4}-\d{2}-\d{2}$/);
  });
});
