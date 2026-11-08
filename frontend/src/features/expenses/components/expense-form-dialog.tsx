import * as React from 'react';
import { CircleDollarSign } from 'lucide-react';

import { Alert } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select } from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { ApiError } from '@/lib/api-client';
import { createExpense, updateExpense } from '../api';
import { EXPENSE_METHOD, todayInput } from '../format';
import type { CreateExpenseRequest, Expense, ExpenseCategory, ExpenseMethod, UpdateExpenseRequest } from '../types';

interface ExpenseFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  categories: ExpenseCategory[];
  expense?: Expense | null;
  onSuccess: (saved: Expense) => void;
}

const METHODS = Object.keys(EXPENSE_METHOD) as ExpenseMethod[];

export function ExpenseFormDialog({
  open,
  onOpenChange,
  categories,
  expense,
  onSuccess,
}: ExpenseFormDialogProps) {
  const [categoryId, setCategoryId] = React.useState('');
  const [description, setDescription] = React.useState('');
  const [amount, setAmount] = React.useState('');
  const [expenseDate, setExpenseDate] = React.useState(todayInput());
  const [method, setMethod] = React.useState<ExpenseMethod>('cash');
  const [notes, setNotes] = React.useState('');
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = React.useState<Record<string, string>>({});

  React.useEffect(() => {
    if (!open) return;
    setCategoryId(expense?.category_id ?? categories[0]?.id ?? '');
    setDescription(expense?.description ?? '');
    setAmount(expense ? String(expense.amount) : '');
    setExpenseDate(expense?.expense_date ?? todayInput());
    setMethod(expense?.method ?? 'cash');
    setNotes(expense?.notes ?? '');
    setError(null);
    setFieldErrors({});
    setLoading(false);
    // Categories are intentionally excluded: reloading them must not wipe user input.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, expense]);

  React.useEffect(() => {
    if (!open) return;
    if (!categoryId && categories.length > 0) {
      setCategoryId(categories[0].id);
    }
  }, [open, categories, categoryId]);

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setLoading(true);
    setError(null);
    setFieldErrors({});

    const numericAmount = Number(amount) || 0;
    if (!categoryId) {
      setError('Pilih kategori pengeluaran terlebih dahulu.');
      setLoading(false);
      return;
    }
    if (!description.trim()) {
      setError('Deskripsi pengeluaran wajib diisi.');
      setLoading(false);
      return;
    }
    if (numericAmount <= 0) {
      setError('Nominal pengeluaran harus lebih dari 0.');
      setLoading(false);
      return;
    }

    try {
      let saved: Expense;
      if (expense) {
        const payload: UpdateExpenseRequest = {
          category_id: categoryId,
          description: description.trim(),
          amount: numericAmount,
          expense_date: expenseDate,
          method,
          notes: notes.trim() || null,
        };
        saved = await updateExpense(expense.id, payload);
      } else {
        const payload: CreateExpenseRequest = {
          category_id: categoryId,
          description: description.trim(),
          amount: numericAmount,
          expense_date: expenseDate,
          method,
          notes: notes.trim() || null,
        };
        saved = await createExpense(payload);
      }
      onSuccess(saved);
      onOpenChange(false);
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        setError(err.message);
        setFieldErrors(err.fieldErrors());
      } else {
        setError('Gagal menyimpan pengeluaran.');
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)} className="max-w-lg">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>{expense ? 'Ubah Pengeluaran' : 'Catat Pengeluaran'}</DialogTitle>
            <DialogDescription>
              Pengeluaran operasional akan masuk ke laporan laba dan laporan pengeluaran outlet aktif.
            </DialogDescription>
          </DialogHeader>

          {error && (
            <div className="mb-4">
              <Alert variant="error">{error}</Alert>
            </div>
          )}

          {categories.length === 0 && (
            <div className="mb-4">
              <Alert variant="info">
                Belum ada kategori pengeluaran. Tambahkan kategori terlebih dahulu melalui tombol Kategori.
              </Alert>
            </div>
          )}

          <div className="space-y-4">
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-1.5">
                <Label htmlFor="expense-category">Kategori *</Label>
                <Select
                  id="expense-category"
                  value={categoryId}
                  onChange={(event) => setCategoryId(event.target.value)}
                  required
                >
                  <option value="">Pilih kategori</option>
                  {categories.map((category) => (
                    <option key={category.id} value={category.id}>
                      {category.name}
                    </option>
                  ))}
                </Select>
                {fieldErrors.category_id && <p className="text-xs text-danger">{fieldErrors.category_id}</p>}
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="expense-date">Tanggal *</Label>
                <Input
                  id="expense-date"
                  type="date"
                  value={expenseDate}
                  onChange={(event) => setExpenseDate(event.target.value)}
                  required
                />
                {fieldErrors.expense_date && <p className="text-xs text-danger">{fieldErrors.expense_date}</p>}
              </div>
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="expense-description">Deskripsi *</Label>
              <Input
                id="expense-description"
                value={description}
                onChange={(event) => setDescription(event.target.value)}
                placeholder="Contoh: pembelian tisu, bensin operasional, token listrik"
                maxLength={200}
                required
              />
              {fieldErrors.description && <p className="text-xs text-danger">{fieldErrors.description}</p>}
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-1.5">
                <Label htmlFor="expense-amount">Nominal *</Label>
                <div className="relative">
                  <CircleDollarSign
                    className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted"
                    aria-hidden="true"
                  />
                  <Input
                    id="expense-amount"
                    type="number"
                    min="1"
                    step="1"
                    inputMode="numeric"
                    value={amount}
                    onChange={(event) => setAmount(event.target.value)}
                    className="pl-9 font-mono"
                    required
                  />
                </div>
                {fieldErrors.amount && <p className="text-xs text-danger">{fieldErrors.amount}</p>}
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="expense-method">Metode *</Label>
                <Select
                  id="expense-method"
                  value={method}
                  onChange={(event) => setMethod(event.target.value as ExpenseMethod)}
                >
                  {METHODS.map((value) => (
                    <option key={value} value={value}>
                      {EXPENSE_METHOD[value]}
                    </option>
                  ))}
                </Select>
                {fieldErrors.method && <p className="text-xs text-danger">{fieldErrors.method}</p>}
              </div>
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="expense-notes">Catatan</Label>
              <Textarea
                id="expense-notes"
                value={notes}
                onChange={(event) => setNotes(event.target.value)}
                placeholder="Keterangan tambahan, nomor nota, atau detail pembayaran"
                rows={3}
                maxLength={1000}
              />
              {fieldErrors.notes && <p className="text-xs text-danger">{fieldErrors.notes}</p>}
            </div>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={loading}>
              Batal
            </Button>
            <Button type="submit" loading={loading} disabled={categories.length === 0}>
              Simpan Pengeluaran
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
