import type { ExpenseMethod } from './types';

export const EXPENSE_METHOD: Record<ExpenseMethod, string> = {
  cash: 'Tunai',
  bank_transfer: 'Transfer Bank',
  qris: 'QRIS',
  other: 'Lainnya',
};

export function expenseMethodLabel(method: string): string {
  return EXPENSE_METHOD[method as ExpenseMethod] ?? method;
}

export function formatExpenseDate(value?: string | null): string {
  if (!value) return '—';
  const [year, month, day] = value.slice(0, 10).split('-').map(Number);
  if (!year || !month || !day) return value;
  return new Date(year, month - 1, day).toLocaleDateString('id-ID', {
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  });
}

export function todayInput(): string {
  const now = new Date();
  const year = now.getFullYear();
  const month = String(now.getMonth() + 1).padStart(2, '0');
  const day = String(now.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}
