import * as React from 'react';

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
import { deleteCredential, setCredential } from '../api';

interface CredentialFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  orderId: string;
  hasCredential: boolean;
  onSuccess: () => void;
}

export function CredentialFormDialog({
  open,
  onOpenChange,
  orderId,
  hasCredential,
  onSuccess,
}: CredentialFormDialogProps) {
  const [type, setType] = React.useState<'pin' | 'password' | 'pattern'>('pin');
  const [value, setValue] = React.useState('');
  const [reason, setReason] = React.useState('');
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  React.useEffect(() => {
    if (open) {
      setType('pin');
      setValue('');
      setReason('');
      setError(null);
    }
  }, [open]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!value.trim()) {
      setError('Nilai PIN / Password / Pola wajib diisi.');
      return;
    }
    setLoading(true);
    setError(null);
    try {
      await setCredential(orderId, {
        type,
        value: value.trim(),
        reason: reason.trim() || undefined,
      });
      onSuccess();
      onOpenChange(false);
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError('Gagal menyimpan kredensial perangkat.');
      }
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async () => {
    if (!window.confirm('Yakin ingin menghapus kredensial perangkat ini?')) return;
    setLoading(true);
    setError(null);
    try {
      await deleteCredential(orderId);
      onSuccess();
      onOpenChange(false);
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError('Gagal menghapus kredensial.');
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)} className="max-w-md">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>{hasCredential ? 'Ubah Kredensial Perangkat' : 'Simpan Kredensial Perangkat'}</DialogTitle>
            <DialogDescription>
              Kredensial disimpan dalam bentuk terenkripsi AES-256 dan hanya dapat dibuka oleh
              teknisi yang bertugas atau owner.
            </DialogDescription>
          </DialogHeader>

          {error && (
            <div className="mb-4">
              <Alert variant="error">{error}</Alert>
            </div>
          )}

          <div className="space-y-4 py-2">
            <div>
              <Label htmlFor="cred-type">Jenis Kredensial</Label>
              <Select
                id="cred-type"
                value={type}
                onChange={(e) => setType(e.target.value as 'pin' | 'password' | 'pattern')}
                className="mt-1"
              >
                <option value="pin">PIN Angka</option>
                <option value="password">Password / Kata Sandi</option>
                <option value="pattern">Pola (Contoh: 1-2-5-8-9)</option>
              </Select>
            </div>

            <div>
              <Label htmlFor="cred-val">
                Nilai Kredensial <span className="text-danger">*</span>
              </Label>
              <Input
                id="cred-val"
                type={type === 'password' ? 'password' : 'text'}
                value={value}
                onChange={(e) => setValue(e.target.value)}
                placeholder={
                  type === 'pin' ? 'Contoh: 123456' : type === 'pattern' ? 'Contoh: 1-2-3-6-9' : 'Kata sandi HP'
                }
                required
                className="mt-1 font-mono"
              />
            </div>

            <div>
              <Label htmlFor="cred-reason">Catatan / Alasan Perubahan (Opsional)</Label>
              <Textarea
                id="cred-reason"
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                placeholder="Misal: Pelanggan baru saja mengganti PIN"
                className="mt-1"
                rows={2}
              />
            </div>
          </div>

          <DialogFooter className="flex justify-between items-center">
            {hasCredential ? (
              <Button
                type="button"
                variant="destructive"
                onClick={handleDelete}
                disabled={loading}
                className="mr-auto"
              >
                Hapus
              </Button>
            ) : <span />}
            <div className="flex items-center gap-2">
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={loading}>
                Batal
              </Button>
              <Button type="submit" loading={loading}>
                Simpan
              </Button>
            </div>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
