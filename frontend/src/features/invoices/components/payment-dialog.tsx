import * as React from 'react';
import { Banknote, CircleDollarSign, Wallet } from 'lucide-react';

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
import { createPayment, newIdempotencyKey } from '../api';
import { PAYMENT_METHOD } from '../format';
import type { CreatePaymentRequest, InvoiceDetail, PaymentMethod, PaymentMutation } from '../types';

interface PaymentDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  invoice: InvoiceDetail;
  onSuccess: (result: PaymentMutation) => void;
}

function nowInputValue(): string {
  const now = new Date();
  const pad = (value: number) => String(value).padStart(2, '0');
  return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}T${pad(now.getHours())}:${pad(now.getMinutes())}`;
}

const METHODS: PaymentMethod[] = ['cash', 'bank_transfer', 'qris', 'card', 'other'];

export function PaymentDialog({ open, onOpenChange, invoice, onSuccess }: PaymentDialogProps) {
  const [method, setMethod] = React.useState<PaymentMethod>('cash');
  const [amount, setAmount] = React.useState('');
  const [paidAt, setPaidAt] = React.useState('');
  const [referenceNo, setReferenceNo] = React.useState('');
  const [notes, setNotes] = React.useState('');
  const [idempotencyKey, setIdempotencyKey] = React.useState('');
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  React.useEffect(() => {
    if (!open) return;
    setMethod('cash');
    setAmount(String(invoice.remaining_amount));
    setPaidAt(nowInputValue());
    setReferenceNo('');
    setNotes('');
    setError(null);
    setLoading(false);
    setIdempotencyKey(newIdempotencyKey());
  }, [open, invoice.id, invoice.remaining_amount]);

  const numericAmount = Number(amount) || 0;
  const isOverpay = numericAmount > invoice.remaining_amount;

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    if (numericAmount <= 0) {
      setError('Nominal pembayaran harus lebih dari 0.');
      return;
    }
    if (isOverpay) {
      setError('Nominal pembayaran tidak boleh melebihi sisa tagihan.');
      return;
    }
    if (!idempotencyKey) {
      setError('Sesi pembayaran belum siap. Tutup lalu buka kembali dialog ini.');
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const payload: CreatePaymentRequest = {
        method,
        amount: numericAmount,
        paid_at: paidAt ? new Date(paidAt).toISOString() : null,
        reference_no: method === 'cash' ? null : referenceNo.trim() || null,
        notes: notes.trim() || null,
      };
      const result = await createPayment(invoice.id, payload, idempotencyKey);
      onSuccess(result);
      onOpenChange(false);
    } catch (err: unknown) {
      setError(err instanceof ApiError ? err.message : 'Gagal menyimpan pembayaran.');
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
              <Banknote className="h-5 w-5 text-accent-600" aria-hidden="true" />
              Terima Pembayaran
            </DialogTitle>
            <DialogDescription>
              Invoice {invoice.invoice_no} · {invoice.customer_name} · Tiket {invoice.ticket_no}
            </DialogDescription>
          </DialogHeader>

          {error && (
            <div className="mb-4">
              <Alert variant="error">{error}</Alert>
            </div>
          )}

          <div className="mb-4 grid grid-cols-3 gap-2 rounded-control border border-border bg-background p-3 text-center">
            <div>
              <p className="text-[11px] uppercase tracking-wide text-muted">Total</p>
              <p className="text-sm font-semibold text-ink">{formatCurrency(invoice.total_amount)}</p>
            </div>
            <div>
              <p className="text-[11px] uppercase tracking-wide text-muted">Dibayar</p>
              <p className="text-sm font-semibold text-emerald-700">{formatCurrency(invoice.paid_amount)}</p>
            </div>
            <div>
              <p className="text-[11px] uppercase tracking-wide text-muted">Sisa</p>
              <p className="text-sm font-semibold text-amber-700">{formatCurrency(invoice.remaining_amount)}</p>
            </div>
          </div>

          <div className="space-y-4">
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-1.5">
                <Label htmlFor="payment-method">Metode Pembayaran *</Label>
                <Select
                  id="payment-method"
                  value={method}
                  onChange={(event) => setMethod(event.target.value as PaymentMethod)}
                >
                  {METHODS.map((value) => (
                    <option key={value} value={value}>
                      {PAYMENT_METHOD[value].label}
                    </option>
                  ))}
                </Select>
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="payment-paid-at">Waktu Pembayaran</Label>
                <Input
                  id="payment-paid-at"
                  type="datetime-local"
                  value={paidAt}
                  onChange={(event) => setPaidAt(event.target.value)}
                />
              </div>
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="payment-amount">
                Nominal <span className="text-danger">*</span>
              </Label>
              <div className="relative">
                <CircleDollarSign
                  className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted"
                  aria-hidden="true"
                />
                <Input
                  id="payment-amount"
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
              <div className="flex flex-wrap items-center gap-2">
                <button
                  type="button"
                  onClick={() => setAmount(String(invoice.remaining_amount))}
                  className="rounded-full border border-border px-2.5 py-1 text-[11px] font-medium text-ink hover:bg-background"
                >
                  Bayar penuh
                </button>
                {invoice.remaining_amount > 1 && (
                  <button
                    type="button"
                    onClick={() => setAmount(String(Math.ceil(invoice.remaining_amount / 2)))}
                    className="rounded-full border border-border px-2.5 py-1 text-[11px] font-medium text-ink hover:bg-background"
                  >
                    DP 50%
                  </button>
                )}
                {numericAmount > 0 && (
                  <span className="text-xs text-muted">
                    Preview: <span className="font-medium text-ink">{formatCurrency(numericAmount)}</span>
                  </span>
                )}
              </div>
              {isOverpay && (
                <p className="text-xs text-danger">
                  Melebihi sisa tagihan sebesar {formatCurrency(numericAmount - invoice.remaining_amount)}.
                </p>
              )}
            </div>

            {method !== 'cash' && (
              <div className="space-y-1.5">
                <Label htmlFor="payment-reference">Nomor Referensi</Label>
                <Input
                  id="payment-reference"
                  value={referenceNo}
                  onChange={(event) => setReferenceNo(event.target.value)}
                  placeholder={method === 'qris' ? 'No. referensi QRIS / merchant' : 'No. referensi transfer / kartu'}
                  maxLength={80}
                />
              </div>
            )}

            <div className="space-y-1.5">
              <Label htmlFor="payment-notes">Catatan</Label>
              <Textarea
                id="payment-notes"
                value={notes}
                onChange={(event) => setNotes(event.target.value)}
                placeholder="Contoh: DP 50%, pelunasan, atau keterangan lain"
                rows={2}
                maxLength={1000}
              />
            </div>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={loading}>
              Batal
            </Button>
            <Button
              type="submit"
              loading={loading}
              disabled={numericAmount <= 0 || isOverpay || invoice.remaining_amount <= 0}
              className="gap-2"
            >
              <Wallet className="h-4 w-4" aria-hidden="true" />
              Simpan Pembayaran
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
