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
import { getTechnicians, updateServiceOrder } from '../api';
import type { ServiceOrderDetail, TechnicianOption } from '../types';

interface ServiceEditDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  order: ServiceOrderDetail;
  onSuccess: (updated: ServiceOrderDetail) => void;
}

export function ServiceEditDialog({
  open,
  onOpenChange,
  order,
  onSuccess,
}: ServiceEditDialogProps) {
  const [complaint, setComplaint] = React.useState('');
  const [completeness, setCompleteness] = React.useState('');
  const [physicalCondition, setPhysicalCondition] = React.useState('');
  const [estimatedCost, setEstimatedCost] = React.useState<number | string>(0);
  const [estimatedDoneAt, setEstimatedDoneAt] = React.useState('');
  const [assignedTechnicianId, setAssignedTechnicianId] = React.useState('');
  const [priority, setPriority] = React.useState<'normal' | 'express'>('normal');
  const [notes, setNotes] = React.useState('');
  const [internalNotes, setInternalNotes] = React.useState('');

  const [technicians, setTechnicians] = React.useState<TechnicianOption[]>([]);
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  React.useEffect(() => {
    if (open) {
      setComplaint(order.complaint || '');
      setCompleteness(order.completeness || '');
      setPhysicalCondition(order.physical_condition || '');
      setEstimatedCost(order.estimated_cost || 0);
      setEstimatedDoneAt(order.estimated_done_at ? order.estimated_done_at.slice(0, 16) : '');
      setAssignedTechnicianId(order.assigned_technician?.id || '');
      setPriority(order.priority || 'normal');
      setNotes(order.notes || '');
      setInternalNotes(order.internal_notes || '');
      setError(null);

      getTechnicians()
        .then(setTechnicians)
        .catch((err) => console.error('Failed loading technicians', err));
    }
  }, [open, order]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);
    try {
      const updated = await updateServiceOrder(order.id, {
        complaint: complaint.trim(),
        completeness: completeness.trim() || undefined,
        physical_condition: physicalCondition.trim() || undefined,
        estimated_cost: Number(estimatedCost) || 0,
        estimated_done_at: estimatedDoneAt ? new Date(estimatedDoneAt).toISOString() : null,
        assigned_technician_id: assignedTechnicianId || null,
        priority,
        notes: notes.trim() || undefined,
        internal_notes: internalNotes.trim() || undefined,
      });
      onSuccess(updated);
      onOpenChange(false);
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError('Gagal memperbarui data servis.');
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
            <DialogTitle>Ubah Data Penerimaan Servis</DialogTitle>
            <DialogDescription>
              Edit informasi keluhan, estimasi, teknisi, dan catatan untuk tiket {order.ticket_no}.
            </DialogDescription>
          </DialogHeader>

          {error && (
            <div className="mb-4">
              <Alert variant="error">{error}</Alert>
            </div>
          )}

          <div className="space-y-4 py-2 max-h-[60vh] overflow-y-auto px-1">
            <div>
              <Label htmlFor="edit-complaint">
                Keluhan Utama <span className="text-danger">*</span>
              </Label>
              <Textarea
                id="edit-complaint"
                value={complaint}
                onChange={(e) => setComplaint(e.target.value)}
                required
                className="mt-1"
                rows={2}
              />
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <Label htmlFor="edit-completeness">Kelengkapan</Label>
                <Input
                  id="edit-completeness"
                  value={completeness}
                  onChange={(e) => setCompleteness(e.target.value)}
                  placeholder="Unit HP, casing"
                  className="mt-1"
                />
              </div>
              <div>
                <Label htmlFor="edit-condition">Kondisi Fisik</Label>
                <Input
                  id="edit-condition"
                  value={physicalCondition}
                  onChange={(e) => setPhysicalCondition(e.target.value)}
                  placeholder="Layar retak halus"
                  className="mt-1"
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <Label htmlFor="edit-cost">Estimasi Biaya (Rp)</Label>
                <Input
                  id="edit-cost"
                  type="number"
                  min="0"
                  step="1000"
                  value={estimatedCost}
                  onChange={(e) => setEstimatedCost(e.target.value)}
                  className="mt-1"
                />
              </div>
              <div>
                <Label htmlFor="edit-priority">Prioritas</Label>
                <Select
                  id="edit-priority"
                  value={priority}
                  onChange={(e) => setPriority(e.target.value as 'normal' | 'express')}
                  className="mt-1"
                >
                  <option value="normal">Normal</option>
                  <option value="express">Express</option>
                </Select>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <Label htmlFor="edit-technician">Teknisi</Label>
                <Select
                  id="edit-technician"
                  value={assignedTechnicianId}
                  onChange={(e) => setAssignedTechnicianId(e.target.value)}
                  className="mt-1"
                >
                  <option value="">Belum Ditugaskan</option>
                  {technicians.map((t) => (
                    <option key={t.id} value={t.id}>
                      {t.name}
                    </option>
                  ))}
                </Select>
              </div>
              <div>
                <Label htmlFor="edit-done">Estimasi Selesai</Label>
                <Input
                  id="edit-done"
                  type="datetime-local"
                  value={estimatedDoneAt}
                  onChange={(e) => setEstimatedDoneAt(e.target.value)}
                  className="mt-1"
                />
              </div>
            </div>

            <div>
              <Label htmlFor="edit-notes">Catatan Pelanggan</Label>
              <Textarea
                id="edit-notes"
                value={notes}
                onChange={(e) => setNotes(e.target.value)}
                className="mt-1"
                rows={2}
              />
            </div>

            <div>
              <Label htmlFor="edit-internal">Catatan Internal Teknisi</Label>
              <Textarea
                id="edit-internal"
                value={internalNotes}
                onChange={(e) => setInternalNotes(e.target.value)}
                className="mt-1"
                rows={2}
              />
            </div>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={loading}>
              Batal
            </Button>
            <Button type="submit" loading={loading}>
              Simpan Perubahan
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
