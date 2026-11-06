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
import { adjustStock } from '../api';
import type { Sparepart } from '../types';

interface AdjustmentDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  sparepart: Sparepart;
  onSuccess: () => void;
}

export function AdjustmentDialog({ open, onOpenChange, sparepart, onSuccess }: AdjustmentDialogProps) {
  const [newStock, setNewStock] = React.useState(String(sparepart.stock));
  const [reason, setReason] = React.useState('');
  const [notes, setNotes] = React.useState('');
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  React.useEffect(() => {
    if (!open) return;
    setNewStock(String(sparepart.stock));
    setReason('');
    setNotes('');
    setError(null);
  }, [open, sparepart]);

  const delta = (Number(newStock) || 0) - sparepart.stock;

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setLoading(true);
    setError(null);
    try {
      await adjustStock({
        sparepart_id: sparepart.id,
        new_stock: Number(newStock) || 0,
        reason: reason.trim(),
        notes: notes.trim() || undefined,
      });
      onSuccess();
      onOpenChange(false);
    } catch (err: unknown) {
      setError(err instanceof ApiError ? err.message : 'Gagal menyimpan penyesuaian stok.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)}>
        <DialogHeader>
          <DialogTitle>Penyesuaian Stok</DialogTitle>
          <DialogDescription>
            {sparepart.name} · stok saat ini {sparepart.stock}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          {error && <Alert variant="error">{error}</Alert>}
          <div className="space-y-1.5">
            <Label htmlFor="adjust-new-stock">Stok Fisik Baru *</Label>
            <Input
              id="adjust-new-stock"
              type="number"
              min="0"
              value={newStock}
              onChange={(e) => setNewStock(e.target.value)}
              required
            />
            <p className="text-xs text-muted">
              Selisih: {delta > 0 ? `+${delta}` : delta} unit
            </p>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="adjust-reason">Alasan *</Label>
            <Input
              id="adjust-reason"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="Contoh: Hasil opname mingguan"
              required
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="adjust-notes">Catatan</Label>
            <Textarea id="adjust-notes" rows={2} value={notes} onChange={(e) => setNotes(e.target.value)} />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Batal
            </Button>
            <Button type="submit" loading={loading} disabled={delta === 0}>
              Simpan Penyesuaian
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
