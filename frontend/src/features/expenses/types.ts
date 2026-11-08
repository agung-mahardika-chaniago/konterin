export type ExpenseMethod = 'cash' | 'bank_transfer' | 'qris' | 'other';

export interface ExpenseCategory {
  id: string;
  name: string;
  is_system: boolean;
  expense_count: number;
}

export interface Expense {
  id: string;
  expense_no: string;
  category_id: string;
  category_name: string;
  outlet_id: string;
  outlet_name: string;
  description: string;
  amount: number;
  expense_date: string;
  method: ExpenseMethod;
  receipt_path?: string | null;
  notes?: string | null;
  created_at: string;
  updated_at: string;
}

export interface ExpenseListData {
  items: Expense[];
  total_amount: number;
}

export interface ExpenseFilter {
  q?: string;
  category_id?: string;
  outlet_id?: string;
  from?: string;
  to?: string;
  page?: number;
  per_page?: number;
}

export interface CreateExpenseRequest {
  category_id: string;
  description: string;
  amount: number;
  expense_date: string;
  method: ExpenseMethod;
  notes?: string | null;
}

export interface UpdateExpenseRequest {
  category_id?: string;
  description?: string;
  amount?: number;
  expense_date?: string;
  method?: ExpenseMethod;
  notes?: string | null;
}
