import * as React from 'react';
import { BadgePercent } from 'lucide-react';

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
import { formatCurrency } from '@/features/dashboard/format';
import { ApiError } from '@/lib/api-client';
import { updateInvoice } from '../api';
import type { InvoiceDetail, UpdateInvoiceRequest } from '../types';

interface InvoiceDiscountDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  invoice: InvoiceDetail;
  onSuccess: (invoice: InvoiceDetail) => void;
}

function toDateInput(value?: string | null): string {
  if (!value) return '';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '';
  const pad = (part: number) => String(part).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
}

function dueDateToISO(value: string): string | null {
  if (!value) return null;
  const [year, month, day] = value.split('-').map(Number);
  if (!year || !month || !day) return null;
  return new Date(year, month - 1, day, 23, 59, 59).toISOString();
}

export function InvoiceDiscountDialog({ open, onOpenChange, invoice, onSuccess }: InvoiceDiscountDialogProps) {
  const [discountType, setDiscountType] = React.useState<'amount' | 'percent'>('amount');
  const [discountValue, setDiscountValue] = React.useState('0');
  const [notes, setNotes] = React.useState('');
  const [dueAt, setDueAt] = React.useState('');
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  React.useEffect(() => {
    if (!open) return;
    setDiscountType(invoice.discount_type === 'percent' ? 'percent' : 'amount');
    setDiscountValue(String(invoice.discount_value));
    setNotes(invoice.notes ?? '');
    setDueAt(toDateInput(invoice.due_at));
    setError(null);
    setLoading(false);
  }, [open, invoice]);

  const subtotal = invoice.subtotal_service + invoice.subtotal_part;
  const numericValue = Number(discountValue) || 0;
  const previewDiscount =
    discountType === 'percent' ? Math.round((subtotal * numericValue) / 100) : numericValue;
  const previewTotal = Math.max(subtotal - previewDiscount, 0);

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    if (numericValue < 0) {
      setError('Nilai diskon tidak boleh negatif.');
      return;
    }
    if (discountType === 'percent' && numericValue > 100) {
      setError('Diskon persen maksimal 100.');
      return;
    }
    if (discountType === 'amount' && numericValue > subtotal) {
      setError('Diskon nominal tidak boleh melebihi subtotal.');
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const payload: UpdateInvoiceRequest = {
        discount_type: discountType,
        discount_value: numericValue,
        notes: notes.trim() || null,
        due_at: dueDateToISO(dueAt),
      };
      const updated = await updateInvoice(invoice.id, payload);
      onSuccess(updated);
      onOpenChange(false);
    } catch (err: unknown) {
      setError(err instanceof ApiError ? err.message : 'Gagal menyimpan diskon invoice.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)} className="max-w-lg">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <BadgePercent className="h-5 w-5 text-accent-600" aria-hidden="true" />
              Diskon & Catatan Invoice
            </DialogTitle>
            <DialogDescription>
              Invoice {invoice.invoice_no} · Subtotal {formatCurrency(subtotal)}
            </DialogDescription>
          </DialogHeader>

          {error && (
            <div className="mb-4">
              <Alert variant="error">{error}</Alert>
            </div>
          )}

          <Alert variant="info" className="mb-4">
            Diskon hanya dapat diubah selama invoice masih draft. Total akhir selalu dihitung ulang oleh server.
          </Alert>

          <div className="space-y-4">
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-1.5">
                <Label htmlFor="discount-type">Tipe Diskon</Label>
                <Select
                  id="discount-type"
                  value={discountType}
                  onChange={(event) => setDiscountType(event.target.value as 'amount' | 'percent')}
                >
                  <option value="amount">Nominal (Rp)</option>
                  <option value="percent">Persen (%)</option>
                </Select>
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="discount-value">
                  Nilai Diskon {discountType === 'percent' ? '(%)' : '(Rp)'}
                </Label>
                <Input
                  id="discount-value"
                  type="number"
                  min="0"
                  step="1"
                  max={discountType === 'percent' ? 100 : subtotal}
                  value={discountValue}
                  onChange={(event) => setDiscountValue(event.target.value)}
                  className="font-mono"
                />
              </div>
            </div>

            {discountType === 'percent' && (
              <div className="flex flex-wrap gap-2">
                {[5, 10, 15, 20].map((value) => (
                  <button
                    key={value}
                    type="button"
                    onClick={() => setDiscountValue(String(value))}
                    className="rounded-full border border-border px-2.5 py-1 text-[11px] font-medium text-ink hover:bg-background"
                  >
                    {value}%
                  </button>
                ))}
              </div>
            )}

            <div className="space-y-1.5">
              <Label htmlFor="discount-due-at">Jatuh Tempo (opsional)</Label>
              <Input
                id="discount-due-at"
                type="date"
                value={dueAt}
                onChange={(event) => setDueAt(event.target.value)}
              />
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="discount-notes">Catatan Invoice</Label>
              <Textarea
                id="discount-notes"
                value={notes}
                onChange={(event) => setNotes(event.target.value)}
                placeholder="Catatan yang tampil pada invoice, misalnya kesepakatan harga"
                rows={3}
                maxLength={2000}
              />
            </div>

            <div className="rounded-control border border-border bg-background p-3 text-sm">
              <div className="flex justify-between py-0.5">
                <span className="text-muted">Subtotal</span>
                <span className="font-medium text-ink">{formatCurrency(subtotal)}</span>
              </div>
              <div className="flex justify-between py-0.5">
                <span className="text-muted">Diskon</span>
                <span className="font-medium text-danger">- {formatCurrency(previewDiscount)}</span>
              </div>
              <div className="mt-1 flex justify-between border-t border-border pt-2 font-semibold">
                <span>Total setelah diskon</span>
                <span className="font-mono">{formatCurrency(previewTotal)}</span>
              </div>
            </div>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={loading}>
              Batal
            </Button>
            <Button type="submit" loading={loading}>
              Simpan Diskon
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
