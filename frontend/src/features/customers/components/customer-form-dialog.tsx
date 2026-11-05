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
import { createCustomer, updateCustomer } from '../api';
import type { Customer } from '../types';

interface CustomerFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  customer?: Customer | null;
  onSuccess: (saved: Customer) => void;
}

export function CustomerFormDialog({
  open,
  onOpenChange,
  customer,
  onSuccess,
}: CustomerFormDialogProps) {
  const [name, setName] = React.useState('');
  const [phone, setPhone] = React.useState('');
  const [address, setAddress] = React.useState('');
  const [notes, setNotes] = React.useState('');
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = React.useState<Record<string, string>>({});

  React.useEffect(() => {
    if (customer) {
      setName(customer.name || '');
      setPhone(customer.phone || '');
      setAddress(customer.address || '');
      setNotes(customer.notes || '');
    } else {
      setName('');
      setPhone('');
      setAddress('');
      setNotes('');
    }
    setError(null);
    setFieldErrors({});
  }, [customer, open]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);
    setFieldErrors({});

    try {
      let result: Customer;
      if (customer) {
        result = await updateCustomer(customer.id, {
          name,
          phone,
          address: address.trim() || undefined,
          notes: notes.trim() || undefined,
        });
      } else {
        result = await createCustomer({
          name,
          phone,
          address: address.trim() || undefined,
          notes: notes.trim() || undefined,
        });
      }
      onSuccess(result);
      onOpenChange(false);
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message);
        setFieldErrors(err.fieldErrors());
      } else {
        setError('Terjadi kesalahan saat menyimpan data pelanggan.');
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
            <DialogTitle>{customer ? 'Ubah Data Pelanggan' : 'Tambah Pelanggan Baru'}</DialogTitle>
            <DialogDescription>
              {customer
                ? 'Perbarui informasi kontak dan alamat pelanggan.'
                : 'Daftarkan data pelanggan baru ke sistem.'}
            </DialogDescription>
          </DialogHeader>

          {error && (
            <div className="mb-4">
              <Alert variant="error">{error}</Alert>
            </div>
          )}

          <div className="space-y-4 py-2">
            <div>
              <Label htmlFor="cust-name">
                Nama Pelanggan <span className="text-danger">*</span>
              </Label>
              <Input
                id="cust-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="Contoh: Siti Rahma"
                required
                className="mt-1"
              />
              {fieldErrors.name && <p className="mt-1 text-xs text-danger">{fieldErrors.name}</p>}
            </div>

            <div>
              <Label htmlFor="cust-phone">
                Nomor Telepon / WhatsApp <span className="text-danger">*</span>
              </Label>
              <Input
                id="cust-phone"
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
                placeholder="0812xxxxxxxx atau +62812xxxxxxxx"
                required
                className="mt-1"
              />
              <p className="mt-1 text-[11px] text-muted">Nomor akan dinormalisasi ke format 628xxx secara otomatis.</p>
              {fieldErrors.phone && <p className="mt-1 text-xs text-danger">{fieldErrors.phone}</p>}
            </div>

            <div>
              <Label htmlFor="cust-address">Alamat (Opsional)</Label>
              <Input
                id="cust-address"
                value={address}
                onChange={(e) => setAddress(e.target.value)}
                placeholder="Jl. Merdeka No. 10"
                className="mt-1"
              />
              {fieldErrors.address && <p className="mt-1 text-xs text-danger">{fieldErrors.address}</p>}
            </div>

            <div>
              <Label htmlFor="cust-notes">Catatan (Opsional)</Label>
              <Textarea
                id="cust-notes"
                value={notes}
                onChange={(e) => setNotes(e.target.value)}
                placeholder="Catatan tambahan seputar pelanggan..."
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
              {customer ? 'Simpan Perubahan' : 'Tambah Pelanggan'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
