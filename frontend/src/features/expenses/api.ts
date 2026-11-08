import { api } from '@/lib/api-client';
import type { ApiEnvelope } from '@/types/api';
import type {
  CreateExpenseRequest,
  Expense,
  ExpenseCategory,
  ExpenseFilter,
  ExpenseListData,
  UpdateExpenseRequest,
} from './types';

function pageQuery(filter: object): string {
  const params = new URLSearchParams();
  Object.entries(filter).forEach(([key, value]) => {
    if (value !== undefined && value !== null && String(value).trim() !== '') {
      params.set(key, String(value));
    }
  });
  const query = params.toString();
  return query ? `?${query}` : '';
}

export async function listExpenses(filter: ExpenseFilter = {}): Promise<ApiEnvelope<ExpenseListData>> {
  return api.getEnvelope<ExpenseListData>(`/expenses${pageQuery(filter)}`);
}

export async function createExpense(payload: CreateExpenseRequest): Promise<Expense> {
  return api.post<Expense>('/expenses', payload);
}

export async function updateExpense(id: string, payload: UpdateExpenseRequest): Promise<Expense> {
  return api.patch<Expense>(`/expenses/${id}`, payload);
}

export async function deleteExpense(id: string): Promise<void> {
  await api.delete<{ message: string }>(`/expenses/${id}`);
}

export async function listExpenseCategories(): Promise<ExpenseCategory[]> {
  return api.get<ExpenseCategory[]>('/expense-categories');
}

export async function createExpenseCategory(name: string): Promise<ExpenseCategory> {
  return api.post<ExpenseCategory>('/expense-categories', { name });
}
