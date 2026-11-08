import * as React from 'react';
import { ArrowRightCircle } from 'lucide-react';

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
import { Label } from '@/components/ui/label';
import { Select } from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { formatCurrency } from '@/features/dashboard/format';
import { ApiError } from '@/lib/api-client';
import { hasPermission } from '@/lib/permissions';
import { useAuthStore } from '@/stores/auth-store';
import { changeServiceStatus } from '../api';
import { allowedTransitionsFor } from '../status-flow';
import type { ServiceOrderDetail } from '../types';
import { STATUS_CONFIG, StatusBadge } from './status-badge';

interface StatusChangeDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  order: ServiceOrderDetail;
  onSuccess: (updated: ServiceOrderDetail) => void;
}

export function StatusChangeDialog({ open, onOpenChange, order, onSuccess }: StatusChangeDialogProps) {
  const user = useAuthStore((state) => state.user);
  const canCancel = hasPermission(user, 'service.cancel');
  const canOverridePickup = hasPermission(user, 'service.override_unpaid_pickup');
  const targets = React.useMemo(
    () => allowedTransitionsFor(order.status, { role: user?.role, canCancel }),
    [order.status, user?.role, canCancel],
  );

  const [target, setTarget] = React.useState('');
  const [notes, setNotes] = React.useState('');
  const [cancelReason, setCancelReason] = React.useState('');
  const [overrideReason, setOverrideReason] = React.useState('');
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  React.useEffect(() => {
    if (!open) return;
    setTarget(targets[0] ?? '');
    setNotes('');
    setCancelReason('');
    setOverrideReason('');
    setError(null);
  }, [open, targets]);

  const pickupRemaining = order.invoice?.remaining_amount ?? 0;
  const needsPickupOverride = target === 'picked_up' && pickupRemaining > 0;

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!target) return;
    if (target === 'cancelled' && !cancelReason.trim()) {
      setError('Alasan pembatalan wajib diisi.');
      return;
    }
    if (needsPickupOverride && !canOverridePickup) {
      setError('Servis belum lunas. Hanya Owner yang dapat melakukan override pickup belum lunas.');
      return;
    }
    if (needsPickupOverride && !overrideReason.trim()) {
      setError('Alasan override pickup belum lunas wajib diisi.');
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const updated = await changeServiceStatus(order.id, {
        status: target,
        notes: notes.trim() || null,
        cancel_reason: target === 'cancelled' ? cancelReason.trim() : null,
        override_reason: needsPickupOverride && canOverridePickup ? overrideReason.trim() : null,
      });
      onSuccess(updated);
      onOpenChange(false);
    } catch (err: unknown) {
      setError(err instanceof ApiError ? err.message : 'Gagal mengubah status servis.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)} className="max-w-md">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <ArrowRightCircle className="h-5 w-5 text-accent-600" aria-hidden="true" />
              Ubah Status Servis
            </DialogTitle>
            <DialogDescription>
              Perpindahan status divalidasi server. Riwayat akan tercatat lengkap dengan nama Anda.
            </DialogDescription>
          </DialogHeader>

          {error && (
            <div className="mb-4">
              <Alert variant="error">{error}</Alert>
            </div>
          )}

          <div className="space-y-4 py-2">
            <div>
              <Label>Status Saat Ini</Label>
              <div className="mt-1">
                <StatusBadge status={order.status} />
              </div>
            </div>

            {targets.length === 0 ? (
              <Alert variant="info">
                Tidak ada transisi status yang tersedia untuk peran Anda pada status ini.
              </Alert>
            ) : (
              <>
                <div>
                  <Label htmlFor="next-status">Status Berikutnya</Label>
                  <Select
                    id="next-status"
                    value={target}
                    onChange={(event) => setTarget(event.target.value)}
                    className="mt-1"
                  >
                    {targets.map((status) => (
                      <option key={status} value={status}>
                        {STATUS_CONFIG[status]?.label ?? status}
                      </option>
                    ))}
                  </Select>
                </div>

                {target === 'cancelled' && (
                  <div>
                    <Label htmlFor="cancel-reason">
                      Alasan Pembatalan <span className="text-danger">*</span>
                    </Label>
                    <Textarea
                      id="cancel-reason"
                      value={cancelReason}
                      onChange={(event) => setCancelReason(event.target.value)}
                      placeholder="Contoh: pelanggan menolak biaya perbaikan"
                      rows={3}
                      className="mt-1"
                      required
                    />
                  </div>
                )}

                {needsPickupOverride && (
                  <Alert variant={canOverridePickup ? 'info' : 'error'}>
                    <p className="font-semibold">Invoice belum lunas</p>
                    <p className="mt-0.5 text-xs">
                      {order.invoice?.invoice_no ? `Invoice ${order.invoice.invoice_no} · ` : ''}
                      Sisa tagihan {formatCurrency(pickupRemaining)}.
                      {canOverridePickup
                        ? ' Sebagai Owner, Anda dapat melakukan override dengan alasan tertulis.'
                        : ' Selesaikan pembayaran terlebih dahulu atau minta Owner melakukan override.'}
                    </p>
                  </Alert>
                )}

                {needsPickupOverride && canOverridePickup && (
                  <div>
                    <Label htmlFor="override-reason">
                      Alasan Override Pickup <span className="text-danger">*</span>
                    </Label>
                    <Textarea
                      id="override-reason"
                      value={overrideReason}
                      onChange={(event) => setOverrideReason(event.target.value)}
                      placeholder="Contoh: pelanggan janji melunasi besok, unit dibutuhkan mendesak"
                      rows={3}
                      className="mt-1"
                      required
                    />
                    <p className="mt-1 text-xs text-muted">
                      Override dicatat pada audit log dan riwayat status servis.
                    </p>
                  </div>
                )}

                <div>
                  <Label htmlFor="status-notes">Catatan Perubahan (opsional)</Label>
                  <Textarea
                    id="status-notes"
                    value={notes}
                    onChange={(event) => setNotes(event.target.value)}
                    placeholder="Informasi tambahan untuk tim atau pelanggan"
                    rows={2}
                    className="mt-1"
                  />
                </div>
              </>
            )}
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={loading}>
              Batal
            </Button>
            <Button type="submit" loading={loading} disabled={!target}>
              Simpan Status
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
