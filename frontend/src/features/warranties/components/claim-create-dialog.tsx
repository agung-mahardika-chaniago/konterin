import * as React from 'react';
import { ShieldPlus } from 'lucide-react';

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
import { createClaim } from '../api';
import type { Claim, Warranty } from '../types';

export function ClaimCreateDialog({
  open,
  onOpenChange,
  warranty,
  onCreated,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  warranty: Warranty;
  onCreated: (claim: Claim) => void;
}) {
  const [complaint, setComplaint] = React.useState('');
  const [errors, setErrors] = React.useState<Record<string, string>>({});
  const [error, setError] = React.useState<string | null>(null);
  const [loading, setLoading] = React.useState(false);

  React.useEffect(() => {
    if (!open) return;
    setComplaint('');
    setErrors({});
    setError(null);
    setLoading(false);
  }, [open]);

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    setError(null);
    setErrors({});
    const trimmed = complaint.trim();
    if (!trimmed) {
      setErrors({ complaint: 'Keluhan klaim wajib diisi.' });
      return;
    }
    setLoading(true);
    try {
      const claim = await createClaim(warranty.id, { complaint: trimmed });
      onCreated(claim);
      onOpenChange(false);
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        setErrors(err.fieldErrors());
        setError(err.message);
      } else {
        setError('Gagal membuat klaim garansi.');
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)} className="max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <ShieldPlus className="h-5 w-5 text-accent-600" aria-hidden="true" />
            Klaim Garansi
          </DialogTitle>
          <DialogDescription>
            Klaim untuk {warranty.warranty_no} — {warranty.device.brand} {warranty.device.model} ({warranty.customer.name}).
          </DialogDescription>
        </DialogHeader>

        {error && (
          <div className="mb-4">
            <Alert variant="error">{error}</Alert>
          </div>
        )}

        <form onSubmit={submit} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="claim-complaint">Keluhan Pelanggan</Label>
            <Textarea
              id="claim-complaint"
              value={complaint}
              onChange={(event) => setComplaint(event.target.value)}
              placeholder="Jelaskan masalah yang muncul setelah servis sebelumnya."
              rows={4}
              maxLength={2000}
              aria-invalid={Boolean(errors.complaint)}
            />
            {errors.complaint && <p className="text-xs text-danger">{errors.complaint}</p>}
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Batal
            </Button>
            <Button type="submit" loading={loading} className="gap-2">
              <ShieldPlus className="h-4 w-4" aria-hidden="true" />
              Buat Klaim
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
