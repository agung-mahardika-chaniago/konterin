import * as React from 'react';
import { Check, Copy, KeyRound } from 'lucide-react';

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
import { resetUserPassword } from '../api';
import type { ResetPasswordResult, User } from '../types';

interface ResetPasswordDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  user: User | null;
}

/** Password reset result is shown exactly once; the server never returns it again. */
export function ResetPasswordDialog({ open, onOpenChange, user }: ResetPasswordDialogProps) {
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [result, setResult] = React.useState<ResetPasswordResult | null>(null);
  const [copied, setCopied] = React.useState(false);

  React.useEffect(() => {
    if (!open) return;
    setError(null);
    setResult(null);
    setCopied(false);
    setLoading(false);
  }, [open]);

  const handleReset = async () => {
    if (!user) return;
    setLoading(true);
    setError(null);
    try {
      const response = await resetUserPassword(user.id);
      setResult(response);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Gagal mereset kata sandi.');
    } finally {
      setLoading(false);
    }
  };

  const handleCopy = async () => {
    if (!result) return;
    try {
      await navigator.clipboard.writeText(result.temporary_password);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      setCopied(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)} className="max-w-md">
        <DialogHeader>
          <DialogTitle>Reset Kata Sandi</DialogTitle>
          <DialogDescription>
            {user ? `Buat kata sandi sementara baru untuk ${user.name} (${user.email}).` : ''}
          </DialogDescription>
        </DialogHeader>

        {error && (
          <div className="mb-4">
            <Alert variant="error">{error}</Alert>
          </div>
        )}

        {result ? (
          <div className="space-y-4">
            <Alert variant="success">
              Kata sandi sementara berhasil dibuat. Sampaikan kepada pengguna — kata sandi ini hanya ditampilkan
              sekali.
            </Alert>
            <div className="space-y-1.5">
              <p className="text-xs font-medium uppercase tracking-wide text-muted">Kata Sandi Sementara</p>
              <div className="flex items-center gap-2">
                <code className="flex-1 rounded-control border border-border bg-surface px-3 py-2.5 font-mono text-sm text-ink">
                  {result.temporary_password}
                </code>
                <Button type="button" variant="outline" onClick={handleCopy} className="h-10 gap-1.5">
                  {copied ? (
                    <>
                      <Check className="h-4 w-4 text-emerald-600" aria-hidden="true" />
                      Tersalin
                    </>
                  ) : (
                    <>
                      <Copy className="h-4 w-4" aria-hidden="true" />
                      Salin
                    </>
                  )}
                </Button>
              </div>
              <p className="text-xs text-muted">
                Pengguna wajib mengganti kata sandi ini saat login berikutnya.
              </p>
            </div>
          </div>
        ) : (
          <div className="space-y-3">
            <div className="flex items-start gap-3 rounded-control border border-border bg-surface p-3">
              <KeyRound className="mt-0.5 h-5 w-5 text-muted" aria-hidden="true" />
              <p className="text-sm leading-6 text-muted">
                Kata sandi lama akan langsung tidak berlaku. Kata sandi sementara hanya muncul sekali — pastikan
                sudah tersalin sebelum menutup dialog ini.
              </p>
            </div>
            <Button onClick={handleReset} loading={loading} variant="destructive" className="w-full sm:w-auto">
              Reset Sekarang
            </Button>
          </div>
        )}

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Tutup
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
