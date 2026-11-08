import * as React from 'react';
import { Plus, Tags } from 'lucide-react';

import { Alert } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
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
import { ApiError } from '@/lib/api-client';
import { createExpenseCategory } from '../api';
import type { ExpenseCategory } from '../types';

interface ExpenseCategoryDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  categories: ExpenseCategory[];
  canManage: boolean;
  onChanged: () => void | Promise<void>;
}

export function ExpenseCategoryDialog({
  open,
  onOpenChange,
  categories,
  canManage,
  onChanged,
}: ExpenseCategoryDialogProps) {
  const [name, setName] = React.useState('');
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [success, setSuccess] = React.useState<string | null>(null);

  React.useEffect(() => {
    if (!open) return;
    setName('');
    setError(null);
    setSuccess(null);
    setLoading(false);
  }, [open]);

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!name.trim()) {
      setError('Nama kategori wajib diisi.');
      return;
    }
    setLoading(true);
    setError(null);
    setSuccess(null);
    try {
      await createExpenseCategory(name.trim());
      setSuccess(`Kategori "${name.trim()}" berhasil ditambahkan.`);
      setName('');
      await onChanged();
    } catch (err: unknown) {
      setError(err instanceof ApiError ? err.message : 'Gagal menambahkan kategori.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)} className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Tags className="h-5 w-5 text-accent-600" aria-hidden="true" />
            Kategori Pengeluaran
          </DialogTitle>
          <DialogDescription>
            Kategori bawaan sistem tidak dapat diubah. Tambahkan kategori khusus sesuai kebutuhan outlet.
          </DialogDescription>
        </DialogHeader>

        {error && (
          <div className="mb-4">
            <Alert variant="error">{error}</Alert>
          </div>
        )}
        {success && (
          <div className="mb-4">
            <Alert variant="success">{success}</Alert>
          </div>
        )}

        <div className="max-h-56 space-y-1 overflow-y-auto rounded-control border border-border p-2">
          {categories.length === 0 ? (
            <p className="p-2 text-center text-xs text-muted">Belum ada kategori.</p>
          ) : (
            categories.map((category) => (
              <div
                key={category.id}
                className="flex items-center justify-between gap-2 rounded px-2 py-1.5 text-sm hover:bg-background"
              >
                <span className="text-ink">{category.name}</span>
                <div className="flex items-center gap-1.5">
                  {category.is_system && <Badge variant="neutral">Sistem</Badge>}
                  <span className="text-[11px] text-muted">{category.expense_count} pengeluaran</span>
                </div>
              </div>
            ))
          )}
        </div>

        {canManage && (
          <form onSubmit={handleSubmit} className="mt-4 space-y-1.5">
            <Label htmlFor="new-category-name">Kategori Baru</Label>
            <div className="flex gap-2">
              <Input
                id="new-category-name"
                value={name}
                onChange={(event) => setName(event.target.value)}
                placeholder="Contoh: Konsumsi tim"
                maxLength={80}
              />
              <Button type="submit" loading={loading} disabled={!name.trim()} className="shrink-0 gap-1.5">
                <Plus className="h-4 w-4" aria-hidden="true" />
                Tambah
              </Button>
            </div>
          </form>
        )}

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Tutup
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
