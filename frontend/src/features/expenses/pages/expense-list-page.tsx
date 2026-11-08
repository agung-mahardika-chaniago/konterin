import * as React from 'react';
import {
  ChevronLeft,
  ChevronRight,
  Pencil,
  Plus,
  ReceiptText,
  Search,
  Tags,
  Trash2,
} from 'lucide-react';

import { EmptyState } from '@/components/shared/empty-state';
import { Alert } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Select } from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { formatCurrency } from '@/features/dashboard/format';
import { hasPermission } from '@/lib/permissions';
import { useAuthStore } from '@/stores/auth-store';
import { deleteExpense, listExpenseCategories, listExpenses } from '../api';
import { ExpenseCategoryDialog } from '../components/expense-category-dialog';
import { ExpenseFormDialog } from '../components/expense-form-dialog';
import { expenseMethodLabel, formatExpenseDate } from '../format';
import type { Expense, ExpenseCategory } from '../types';

export function ExpenseListPage() {
  const user = useAuthStore((state) => state.user);
  const canCreate = hasPermission(user, 'expense.create');
  const canUpdate = hasPermission(user, 'expense.update');
  const canDelete = hasPermission(user, 'expense.delete');

  const [items, setItems] = React.useState<Expense[]>([]);
  const [total, setTotal] = React.useState(0);
  const [totalAmount, setTotalAmount] = React.useState(0);
  const [page, setPage] = React.useState(1);
  const [totalPages, setTotalPages] = React.useState(1);
  const [search, setSearch] = React.useState('');
  const [debouncedSearch, setDebouncedSearch] = React.useState('');
  const [categoryId, setCategoryId] = React.useState('');
  const [from, setFrom] = React.useState('');
  const [to, setTo] = React.useState('');
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);
  const [success, setSuccess] = React.useState<string | null>(null);

  const [categories, setCategories] = React.useState<ExpenseCategory[]>([]);
  const [formOpen, setFormOpen] = React.useState(false);
  const [editing, setEditing] = React.useState<Expense | null>(null);
  const [categoryOpen, setCategoryOpen] = React.useState(false);

  React.useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search);
      setPage(1);
    }, 300);
    return () => clearTimeout(timer);
  }, [search]);

  const loadCategories = React.useCallback(async () => {
    try {
      const data = await listExpenseCategories();
      setCategories(data);
    } catch {
      setCategories([]);
    }
  }, []);

  const loadData = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await listExpenses({
        q: debouncedSearch.trim() || undefined,
        category_id: categoryId || undefined,
        from: from || undefined,
        to: to || undefined,
        page,
        per_page: 15,
      });
      setItems(result.data?.items ?? []);
      setTotalAmount(result.data?.total_amount ?? 0);
      if (result.meta) {
        setTotal(result.meta.total);
        setTotalPages(result.meta.total_pages || 1);
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Gagal memuat daftar pengeluaran.');
    } finally {
      setLoading(false);
    }
  }, [debouncedSearch, categoryId, from, to, page]);

  React.useEffect(() => {
    loadCategories();
  }, [loadCategories]);

  React.useEffect(() => {
    loadData();
  }, [loadData]);

  const handleSaved = (saved: Expense) => {
    setSuccess(`Pengeluaran ${saved.expense_no} berhasil disimpan.`);
    loadData();
    loadCategories();
  };

  const handleDelete = async (expense: Expense) => {
    if (!window.confirm(`Hapus pengeluaran "${expense.description}" sebesar ${formatCurrency(expense.amount)}?`)) {
      return;
    }
    try {
      await deleteExpense(expense.id);
      setSuccess(`Pengeluaran ${expense.expense_no} berhasil dihapus.`);
      loadData();
      loadCategories();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Gagal menghapus pengeluaran.');
    }
  };

  const openCreate = () => {
    setEditing(null);
    setFormOpen(true);
  };

  const openEdit = (expense: Expense) => {
    setEditing(expense);
    setFormOpen(true);
  };

  const resetFilters = () => {
    setSearch('');
    setCategoryId('');
    setFrom('');
    setTo('');
    setPage(1);
  };

  const hasFilter = Boolean(search || categoryId || from || to);

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-ink sm:text-[28px]">Pengeluaran</h1>
          <p className="mt-1 text-sm text-muted">
            Catat biaya operasional outlet untuk perhitungan laba dan laporan keuangan.
          </p>
        </div>
        <div className="flex flex-wrap gap-2 self-start sm:self-auto">
          <Button variant="outline" onClick={() => setCategoryOpen(true)} className="gap-2">
            <Tags className="h-4 w-4" aria-hidden="true" />
            Kategori
          </Button>
          {canCreate && (
            <Button onClick={openCreate} className="gap-2">
              <Plus className="h-4 w-4" aria-hidden="true" />
              Catat Pengeluaran
            </Button>
          )}
        </div>
      </div>

      {success && <Alert variant="success">{success}</Alert>}
      {error && <Alert variant="error">{error}</Alert>}

      <div className="grid gap-4 sm:grid-cols-3">
        <Card className="p-4">
          <p className="text-xs font-medium uppercase tracking-wide text-muted">Total Pengeluaran</p>
          <p className="mt-1 text-2xl font-bold text-ink">{formatCurrency(totalAmount)}</p>
          <p className="text-xs text-muted">sesuai filter aktif</p>
        </Card>
        <Card className="p-4">
          <p className="text-xs font-medium uppercase tracking-wide text-muted">Jumlah Transaksi</p>
          <p className="mt-1 text-2xl font-bold text-ink">{total}</p>
          <p className="text-xs text-muted">pengeluaran tercatat</p>
        </Card>
        <Card className="p-4">
          <p className="text-xs font-medium uppercase tracking-wide text-muted">Kategori</p>
          <p className="mt-1 text-2xl font-bold text-ink">{categories.length}</p>
          <p className="text-xs text-muted">{categories.filter((category) => category.is_system).length} kategori sistem</p>
        </Card>
      </div>

      <div className="flex flex-col gap-3 lg:flex-row lg:items-center">
        <div className="relative flex-1 lg:max-w-md">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted" aria-hidden="true" />
          <Input
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder="Cari nomor, deskripsi, atau kategori..."
            className="pl-9"
          />
        </div>
        <Select
          value={categoryId}
          onChange={(event) => {
            setCategoryId(event.target.value);
            setPage(1);
          }}
          className="lg:max-w-[200px]"
        >
          <option value="">Semua kategori</option>
          {categories.map((category) => (
            <option key={category.id} value={category.id}>
              {category.name}
            </option>
          ))}
        </Select>
        <div className="flex items-center gap-2">
          <Input
            type="date"
            value={from}
            onChange={(event) => {
              setFrom(event.target.value);
              setPage(1);
            }}
            className="w-[145px]"
            title="Tanggal awal"
          />
          <span className="text-xs text-muted">s.d.</span>
          <Input
            type="date"
            value={to}
            onChange={(event) => {
              setTo(event.target.value);
              setPage(1);
            }}
            className="w-[145px]"
            title="Tanggal akhir"
          />
        </div>
        {hasFilter && (
          <Button variant="ghost" size="sm" onClick={resetFilters}>
            Reset filter
          </Button>
        )}
      </div>

      <Card className="overflow-hidden">
        {loading ? (
          <div className="space-y-3 p-4">
            {[1, 2, 3, 4, 5].map((i) => (
              <div key={i} className="flex items-center justify-between py-2">
                <div className="space-y-1">
                  <Skeleton className="h-4 w-48" />
                  <Skeleton className="h-3 w-32" />
                </div>
                <Skeleton className="h-8 w-24" />
              </div>
            ))}
          </div>
        ) : items.length === 0 ? (
          <EmptyState
            icon={ReceiptText}
            title="Belum ada pengeluaran"
            description={
              hasFilter
                ? 'Tidak ada pengeluaran yang sesuai dengan filter Anda.'
                : 'Catat pengeluaran operasional pertama agar laporan laba lebih akurat.'
            }
            action={
              canCreate ? (
                <Button onClick={openCreate} size="sm" className="gap-1.5">
                  <Plus className="h-4 w-4" aria-hidden="true" />
                  Catat Pengeluaran
                </Button>
              ) : undefined
            }
            className="py-12"
          />
        ) : (
          <>
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead className="border-b border-border bg-background text-xs uppercase tracking-wider text-muted">
                  <tr>
                    <th scope="col" className="px-5 py-3 font-semibold">Tanggal</th>
                    <th scope="col" className="px-5 py-3 font-semibold">Pengeluaran</th>
                    <th scope="col" className="px-5 py-3 font-semibold">Kategori</th>
                    <th scope="col" className="px-5 py-3 font-semibold">Metode</th>
                    <th scope="col" className="px-5 py-3 text-right font-semibold">Nominal</th>
                    <th scope="col" className="px-5 py-3 text-right font-semibold">Aksi</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {items.map((expense) => (
                    <tr key={expense.id} className="transition-colors hover:bg-background/50">
                      <td className="px-5 py-3.5 text-xs text-muted">{formatExpenseDate(expense.expense_date)}</td>
                      <td className="px-5 py-3.5">
                        <p className="font-medium text-ink">{expense.description}</p>
                        <p className="font-mono text-[11px] text-muted">{expense.expense_no}</p>
                        {expense.notes && <p className="mt-0.5 text-[11px] italic text-muted">{expense.notes}</p>}
                      </td>
                      <td className="px-5 py-3.5 text-xs">
                        <Badge variant="neutral">{expense.category_name}</Badge>
                      </td>
                      <td className="px-5 py-3.5 text-xs text-ink">{expenseMethodLabel(expense.method)}</td>
                      <td className="px-5 py-3.5 text-right font-mono text-xs font-semibold text-ink">
                        {formatCurrency(expense.amount)}
                      </td>
                      <td className="whitespace-nowrap px-5 py-3.5 text-right">
                        <div className="flex items-center justify-end gap-1">
                          {canUpdate && (
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => openEdit(expense)}
                              className="h-8 px-2"
                              title="Ubah pengeluaran"
                            >
                              <Pencil className="h-3.5 w-3.5 text-muted hover:text-ink" aria-hidden="true" />
                            </Button>
                          )}
                          {canDelete && (
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => handleDelete(expense)}
                              className="h-8 px-2 hover:text-danger"
                              title="Hapus pengeluaran"
                            >
                              <Trash2 className="h-3.5 w-3.5 text-muted hover:text-danger" aria-hidden="true" />
                            </Button>
                          )}
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            <div className="flex items-center justify-between border-t border-border px-5 py-3 text-xs text-muted">
              <span>
                Menampilkan {items.length} dari total {total} pengeluaran
              </span>
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage((current) => Math.max(1, current - 1))}
                  disabled={page <= 1}
                  className="h-8 px-2"
                >
                  <ChevronLeft className="h-4 w-4" aria-hidden="true" />
                </Button>
                <span className="font-medium text-ink">
                  {page} / {totalPages}
                </span>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage((current) => Math.min(totalPages, current + 1))}
                  disabled={page >= totalPages}
                  className="h-8 px-2"
                >
                  <ChevronRight className="h-4 w-4" aria-hidden="true" />
                </Button>
              </div>
            </div>
          </>
        )}
      </Card>

      <ExpenseFormDialog
        open={formOpen}
        onOpenChange={setFormOpen}
        categories={categories}
        expense={editing}
        onSuccess={handleSaved}
      />

      <ExpenseCategoryDialog
        open={categoryOpen}
        onOpenChange={setCategoryOpen}
        categories={categories}
        canManage={canCreate}
        onChanged={loadCategories}
      />
    </div>
  );
}
