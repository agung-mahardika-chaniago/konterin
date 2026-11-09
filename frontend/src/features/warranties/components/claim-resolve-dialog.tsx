import * as React from 'react';
import { CheckCircle2 } from 'lucide-react';

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
import { Textarea } from '@/components/ui/textarea';
import { ApiError } from '@/lib/api-client';
import { updateClaim } from '../api';
import { claimStatusLabel } from '../format';
import type { Claim, ClaimStatus } from '../types';

export function ClaimResolveDialog({
  open,
  onOpenChange,
  claim,
  targetStatus,
  onUpdated,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  claim: Claim;
  targetStatus: Extract<ClaimStatus, 'completed' | 'rejected'>;
  onUpdated: (claim: Claim) => void;
}) {
  const [resolution, setResolution] = React.useState('');
  const [error, setError] = React.useState<string | null>(null);
  const [loading, setLoading] = React.useState(false);

  React.useEffect(() => {
    if (!open) return;
    setResolution(claim.resolution ?? '');
    setError(null);
    setLoading(false);
  }, [open, claim.resolution]);

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    setError(null);
    if (!resolution.trim()) {
      setError('Resolusi wajib diisi saat klaim selesai atau ditolak.');
      return;
    }
    setLoading(true);
    try {
      const updated = await updateClaim(claim.id, {
        status: targetStatus,
        resolution: resolution.trim(),
      });
      onUpdated(updated);
      onOpenChange(false);
    } catch (err: unknown) {
      setError(err instanceof ApiError ? err.message : 'Gagal memperbarui klaim.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)} className="max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <CheckCircle2 className="h-5 w-5 text-accent-600" aria-hidden="true" />
            {targetStatus === 'completed' ? 'Selesaikan Klaim' : 'Tolak Klaim'}
          </DialogTitle>
          <DialogDescription>
            Klaim {claim.claim_no} akan diubah menjadi status {claimStatusLabel(targetStatus)}. Tuliskan hasil
            penanganan agar riwayat klaim tetap jelas.
          </DialogDescription>
        </DialogHeader>

        {error && (
          <div className="mb-4">
            <Alert variant="error">{error}</Alert>
          </div>
        )}

        <form onSubmit={submit} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="claim-resolution">Resolusi / Catatan Penanganan</Label>
            <Textarea
              id="claim-resolution"
              value={resolution}
              onChange={(event) => setResolution(event.target.value)}
              rows={4}
              maxLength={2000}
              placeholder={
                targetStatus === 'completed'
                  ? 'Contoh: Baterai diganti unit baru, perangkat sudah diuji dan normal.'
                  : 'Contoh: Kerusakan akibat cairan di luar cakupan garansi.'
              }
            />
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Batal
            </Button>
            <Button
              type="submit"
              loading={loading}
              variant={targetStatus === 'rejected' ? 'destructive' : 'default'}
              className="gap-2"
            >
              <CheckCircle2 className="h-4 w-4" aria-hidden="true" />
              {targetStatus === 'completed' ? 'Selesaikan Klaim' : 'Tolak Klaim'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
