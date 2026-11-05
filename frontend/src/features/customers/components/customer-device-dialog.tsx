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
import { createCustomerDevice } from '../api';
import type { Device } from '../types';

interface CustomerDeviceDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  customerId: string;
  customerName: string;
  onSuccess: (newDevice: Device) => void;
}

export function CustomerDeviceDialog({
  open,
  onOpenChange,
  customerId,
  customerName,
  onSuccess,
}: CustomerDeviceDialogProps) {
  const [brand, setBrand] = React.useState('');
  const [model, setModel] = React.useState('');
  const [imei, setImei] = React.useState('');
  const [color, setColor] = React.useState('');
  const [notes, setNotes] = React.useState('');
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = React.useState<Record<string, string>>({});

  React.useEffect(() => {
    if (open) {
      setBrand('');
      setModel('');
      setImei('');
      setColor('');
      setNotes('');
      setError(null);
      setFieldErrors({});
    }
  }, [open]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);
    setFieldErrors({});

    try {
      const created = await createCustomerDevice(customerId, {
        brand,
        model,
        imei: imei.trim() || undefined,
        color: color.trim() || undefined,
        notes: notes.trim() || undefined,
      });
      onSuccess(created);
      onOpenChange(false);
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message);
        setFieldErrors(err.fieldErrors());
      } else {
        setError('Terjadi kesalahan saat menambahkan perangkat.');
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
            <DialogTitle>Tambah Perangkat</DialogTitle>
            <DialogDescription>
              Daftarkan perangkat baru milik pelanggan <span className="font-semibold text-ink">{customerName}</span>.
            </DialogDescription>
          </DialogHeader>

          {error && (
            <div className="mb-4">
              <Alert variant="error">{error}</Alert>
            </div>
          )}

          <div className="space-y-4 py-2">
            <div className="grid grid-cols-2 gap-3">
              <div>
                <Label htmlFor="dev-brand">
                  Merek <span className="text-danger">*</span>
                </Label>
                <Input
                  id="dev-brand"
                  value={brand}
                  onChange={(e) => setBrand(e.target.value)}
                  placeholder="Samsung, iPhone, Xiaomi"
                  required
                  className="mt-1"
                />
                {fieldErrors.brand && <p className="mt-1 text-xs text-danger">{fieldErrors.brand}</p>}
              </div>
              <div>
                <Label htmlFor="dev-model">
                  Model / Tipe <span className="text-danger">*</span>
                </Label>
                <Input
                  id="dev-model"
                  value={model}
                  onChange={(e) => setModel(e.target.value)}
                  placeholder="Galaxy S22, 13 Pro"
                  required
                  className="mt-1"
                />
                {fieldErrors.model && <p className="mt-1 text-xs text-danger">{fieldErrors.model}</p>}
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <Label htmlFor="dev-imei">Nomor Seri / IMEI</Label>
                <Input
                  id="dev-imei"
                  value={imei}
                  onChange={(e) => setImei(e.target.value)}
                  placeholder="15 digit IMEI"
                  className="mt-1"
                />
                {fieldErrors.imei && <p className="mt-1 text-xs text-danger">{fieldErrors.imei}</p>}
              </div>
              <div>
                <Label htmlFor="dev-color">Warna</Label>
                <Input
                  id="dev-color"
                  value={color}
                  onChange={(e) => setColor(e.target.value)}
                  placeholder="Hitam, Putih, Biru"
                  className="mt-1"
                />
                {fieldErrors.color && <p className="mt-1 text-xs text-danger">{fieldErrors.color}</p>}
              </div>
            </div>

            <div>
              <Label htmlFor="dev-notes">Catatan Perangkat</Label>
              <Textarea
                id="dev-notes"
                value={notes}
                onChange={(e) => setNotes(e.target.value)}
                placeholder="Misal: kondisi fisik umum, riwayat servis luar"
                className="mt-1"
                rows={2}
              />
              {fieldErrors.notes && <p className="mt-1 text-xs text-danger">{fieldErrors.notes}</p>}
            </div>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={loading}>
              Batal
            </Button>
            <Button type="submit" loading={loading}>
              Simpan Perangkat
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
