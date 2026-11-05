import * as React from 'react';
import { Eye, KeyRound, ShieldAlert } from 'lucide-react';

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
import { revealCredential } from '../api';

interface CredentialRevealDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  orderId: string;
  ticketNo: string;
}

export function CredentialRevealDialog({
  open,
  onOpenChange,
  orderId,
  ticketNo,
}: CredentialRevealDialogProps) {
  const [reason, setReason] = React.useState('');
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [revealed, setRevealed] = React.useState<{ type: string; value: string } | null>(null);

  React.useEffect(() => {
    if (open) {
      setReason('');
      setError(null);
      setRevealed(null);
    }
  }, [open]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!reason.trim()) {
      setError('Alasan wajib diisi untuk keperluan audit keamanan.');
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const res = await revealCredential(orderId, reason.trim());
      setRevealed(res);
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError('Gagal membuka kredensial perangkat.');
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)} className="max-w-md">
        <DialogHeader>
          <div className="flex items-center gap-2 text-brand-900">
            <KeyRound className="h-5 w-5 text-accent-600" aria-hidden="true" />
            <DialogTitle>Lihat Kredensial Perangkat</DialogTitle>
          </div>
          <DialogDescription>
            Tiket <span className="font-semibold text-ink">{ticketNo}</span>. Akses kredensial
            ini dipantau dan tercatat dalam audit log sistem.
          </DialogDescription>
        </DialogHeader>

        {error && (
          <div className="mb-3">
            <Alert variant="error">{error}</Alert>
          </div>
        )}

        {revealed ? (
          <div className="space-y-4 py-2">
            <div className="rounded-lg border border-border bg-background p-4 text-center">
              <span className="text-xs uppercase font-semibold tracking-wider text-muted block mb-1">
                Tipe Kredensial: {revealed.type.toUpperCase()}
              </span>
              <p className="text-2xl font-mono font-bold tracking-widest text-ink select-all">
                {revealed.value}
              </p>
            </div>
            <p className="text-xs text-muted text-center">
              Catatan: Kredensial ini bersifat rahasia. Jangan berikan kepada pihak luar.
            </p>
            <DialogFooter>
              <Button onClick={() => onOpenChange(false)} className="w-full">
                Selesai
              </Button>
            </DialogFooter>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="space-y-4 py-2">
            <div className="rounded-md bg-amber-50 p-3 text-xs text-amber-900 border border-amber-200 flex items-start gap-2">
              <ShieldAlert className="h-4 w-4 shrink-0 mt-0.5 text-amber-700" aria-hidden="true" />
              <span>
                Demi keamanan data pelanggan, pembukaan PIN/Password memerlukan alasan operasional yang sah.
              </span>
            </div>

            <div>
              <Label htmlFor="reveal-reason">
                Alasan Pembukaan <span className="text-danger">*</span>
              </Label>
              <Textarea
                id="reveal-reason"
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                placeholder="Contoh: Perlu membuka kunci layar untuk verifikasi fungsi touch screen dan kamera."
                required
                className="mt-1"
                rows={3}
              />
            </div>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={loading}>
                Batal
              </Button>
              <Button type="submit" loading={loading} className="gap-2">
                <Eye className="h-4 w-4" aria-hidden="true" />
                Buka Kredensial
              </Button>
            </DialogFooter>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
}
