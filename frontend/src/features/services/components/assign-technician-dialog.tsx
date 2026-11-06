import * as React from 'react';
import { UserCog } from 'lucide-react';

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
import { ApiError } from '@/lib/api-client';
import { assignTechnician, getTechnicians } from '../api';
import type { ServiceOrderDetail, TechnicianOption } from '../types';

interface AssignTechnicianDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  order: ServiceOrderDetail;
  onSuccess: (updated: ServiceOrderDetail) => void;
}

export function AssignTechnicianDialog({ open, onOpenChange, order, onSuccess }: AssignTechnicianDialogProps) {
  const [technicians, setTechnicians] = React.useState<TechnicianOption[]>([]);
  const [technicianId, setTechnicianId] = React.useState('');
  const [priority, setPriority] = React.useState<'normal' | 'express'>('normal');
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  React.useEffect(() => {
    if (!open) return;
    setTechnicianId(order.assigned_technician?.id ?? '');
    setPriority(order.priority ?? 'normal');
    setError(null);
    getTechnicians()
      .then(setTechnicians)
      .catch(() => setTechnicians([]));
  }, [open, order.assigned_technician?.id, order.priority]);

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setLoading(true);
    setError(null);
    try {
      const updated = await assignTechnician(order.id, {
        technician_id: technicianId || null,
        priority,
      });
      onSuccess(updated);
      onOpenChange(false);
    } catch (err: unknown) {
      setError(err instanceof ApiError ? err.message : 'Gagal menugaskan teknisi.');
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
              <UserCog className="h-5 w-5 text-accent-600" aria-hidden="true" />
              Tugaskan Teknisi
            </DialogTitle>
            <DialogDescription>
              Pilih teknisi yang bertanggung jawab dan tentukan prioritas pengerjaan tiket {order.ticket_no}.
            </DialogDescription>
          </DialogHeader>

          {error && (
            <div className="mb-4">
              <Alert variant="error">{error}</Alert>
            </div>
          )}

          <div className="space-y-4 py-2">
            <div>
              <Label htmlFor="assign-technician">Teknisi</Label>
              <Select
                id="assign-technician"
                value={technicianId}
                onChange={(event) => setTechnicianId(event.target.value)}
                className="mt-1"
              >
                <option value="">Belum Ditugaskan</option>
                {technicians.map((technician) => (
                  <option key={technician.id} value={technician.id}>
                    {technician.name}
                  </option>
                ))}
              </Select>
            </div>

            <div>
              <Label htmlFor="assign-priority">Prioritas</Label>
              <Select
                id="assign-priority"
                value={priority}
                onChange={(event) => setPriority(event.target.value as 'normal' | 'express')}
                className="mt-1"
              >
                <option value="normal">Normal</option>
                <option value="express">Express</option>
              </Select>
              {priority === 'express' && (
                <p className="mt-1 text-[11px] text-amber-700">
                  Tiket express akan ditandai dan diprioritaskan pada daftar servis.
                </p>
              )}
            </div>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={loading}>
              Batal
            </Button>
            <Button type="submit" loading={loading}>
              Simpan Penugasan
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
