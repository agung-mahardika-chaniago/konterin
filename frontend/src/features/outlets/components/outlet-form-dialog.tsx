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
import { Textarea } from '@/components/ui/textarea';
import { ApiError } from '@/lib/api-client';
import { createOutlet, updateOutlet } from '../api';
import type { Outlet } from '../types';

interface OutletFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  outlet?: Outlet | null;
  onSuccess: (saved: Outlet, created: boolean) => void;
}

export function OutletFormDialog({ open, onOpenChange, outlet, onSuccess }: OutletFormDialogProps) {
  const isEdit = Boolean(outlet);

  const [code, setCode] = React.useState('');
  const [name, setName] = React.useState('');
  const [address, setAddress] = React.useState('');
  const [phone, setPhone] = React.useState('');
  const [timezone, setTimezone] = React.useState('Asia/Jakarta');
  const [isActive, setIsActive] = React.useState(true);
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = React.useState<Record<string, string>>({});

  React.useEffect(() => {
    if (!open) return;
    setCode(outlet?.code ?? '');
    setName(outlet?.name ?? '');
    setAddress(outlet?.address ?? '');
    setPhone(outlet?.phone ?? '');
    setTimezone(outlet?.timezone ?? 'Asia/Jakarta');
    setIsActive(outlet?.is_active ?? true);
    setError(null);
    setFieldErrors({});
    setLoading(false);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, outlet]);

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setLoading(true);
    setError(null);
    setFieldErrors({});
    const payload = {
      code: code.trim().toUpperCase(),
      name: name.trim(),
      address: address.trim() || null,
      phone: phone.trim() || null,
      timezone: timezone.trim() || 'Asia/Jakarta',
      is_active: isActive,
    };
    try {
      let saved: Outlet;
      if (outlet) {
        saved = await updateOutlet(outlet.id, payload);
        onSuccess(saved, false);
      } else {
        saved = await createOutlet(payload);
        onSuccess(saved, true);
      }
      onOpenChange(false);
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        setError(err.message);
        setFieldErrors(err.fieldErrors());
      } else {
        setError('Gagal menyimpan outlet.');
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)} className="max-w-lg">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>{isEdit ? 'Ubah Outlet' : 'Tambah Outlet'}</DialogTitle>
            <DialogDescription>
              Setiap outlet memiliki stok, nomor dokumen, dan laporan tersendiri.
            </DialogDescription>
          </DialogHeader>

          {error && (
            <div className="mb-4">
              <Alert variant="error">{error}</Alert>
            </div>
          )}

          <div className="space-y-4">
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-1.5">
                <Label htmlFor="outlet-code">Kode *</Label>
                <Input
                  id="outlet-code"
                  value={code}
                  onChange={(event) => setCode(event.target.value)}
                  maxLength={20}
                  minLength={2}
                  required
                  className="font-mono uppercase"
                  placeholder="PST"
                />
                {fieldErrors.code && <p className="text-xs text-danger">{fieldErrors.code}</p>}
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="outlet-name">Nama *</Label>
                <Input
                  id="outlet-name"
                  value={name}
                  onChange={(event) => setName(event.target.value)}
                  maxLength={120}
                  required
                  placeholder="FixPhone Pusat"
                />
                {fieldErrors.name && <p className="text-xs text-danger">{fieldErrors.name}</p>}
              </div>
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="outlet-address">Alamat</Label>
              <Textarea
                id="outlet-address"
                value={address}
                onChange={(event) => setAddress(event.target.value)}
                rows={2}
                maxLength={500}
                placeholder="Jl. ... No. ..., Kota, Kode Pos"
              />
              {fieldErrors.address && <p className="text-xs text-danger">{fieldErrors.address}</p>}
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-1.5">
                <Label htmlFor="outlet-phone">Telepon</Label>
                <Input
                  id="outlet-phone"
                  value={phone}
                  onChange={(event) => setPhone(event.target.value)}
                  maxLength={30}
                  placeholder="08xxxxxxxxxx"
                />
                {fieldErrors.phone && <p className="text-xs text-danger">{fieldErrors.phone}</p>}
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="outlet-timezone">Zona Waktu</Label>
                <Input
                  id="outlet-timezone"
                  list="outlet-timezone-suggestions"
                  value={timezone}
                  onChange={(event) => setTimezone(event.target.value)}
                  maxLength={50}
                  placeholder="Asia/Jakarta"
                />
                <datalist id="outlet-timezone-suggestions">
                  <option value="Asia/Jakarta">WIB — Jakarta</option>
                  <option value="Asia/Makassar">WITA — Makassar</option>
                  <option value="Asia/Jayapura">WIT — Jayapura</option>
                </datalist>
                {fieldErrors.timezone ? (
                  <p className="text-xs text-danger">{fieldErrors.timezone}</p>
                ) : (
                  <p className="text-xs text-muted">Batasi hari untuk nomor dokumen dan laporan outlet ini.</p>
                )}
              </div>
            </div>

            <label htmlFor="outlet-active" className="flex items-start gap-3 rounded-control border border-border bg-surface p-3">
              <input
                id="outlet-active"
                type="checkbox"
                checked={isActive}
                onChange={(event) => setIsActive(event.target.checked)}
                className="mt-0.5 h-4 w-4 rounded border-border text-accent-600 focus:ring-accent-500"
              />
              <span>
                <span className="block text-sm font-medium text-ink">Outlet aktif</span>
                <span className="block text-xs text-muted">
                  Outlet nonaktif disembunyikan dari pilihan operasional baru.
                </span>
              </span>
            </label>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={loading}>
              Batal
            </Button>
            <Button type="submit" loading={loading}>
              {isEdit ? 'Simpan Perubahan' : 'Tambah Outlet'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
