import * as React from 'react';
import { Store } from 'lucide-react';

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
import { ApiError } from '@/lib/api-client';
import { replaceUserOutlets } from '../api';
import type { User, UserOutlet } from '../types';

interface UserOutletsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  outlets: UserOutlet[];
  user: User | null;
  /** Refresh the row data after a successful assignment. */
  onSuccess: (saved: User) => void;
}

export function UserOutletsDialog({ open, onOpenChange, outlets, user, onSuccess }: UserOutletsDialogProps) {
  const [selected, setSelected] = React.useState<string[]>([]);
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = React.useState<Record<string, string>>({});

  React.useEffect(() => {
    if (!open) return;
    setSelected(user?.outlets.map((outlet) => outlet.id) ?? []);
    setError(null);
    setFieldErrors({});
    setLoading(false);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, user]);

  const toggleOutlet = (outletId: string, checked: boolean) => {
    setSelected((current) =>
      checked ? [...current, outletId] : current.filter((id) => id !== outletId),
    );
  };

  const handleSave = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!user) return;
    setLoading(true);
    setError(null);
    setFieldErrors({});
    try {
      const saved = await replaceUserOutlets(user.id, selected);
      onSuccess(saved);
      onOpenChange(false);
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        setError(err.message);
        setFieldErrors(err.fieldErrors());
      } else {
        setError('Gagal menyimpan penugasan outlet.');
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)} className="max-w-md">
        <form onSubmit={handleSave}>
          <DialogHeader>
            <DialogTitle>Penugasan Outlet</DialogTitle>
            <DialogDescription>
              {user ? `Tentukan outlet yang dapat diakses oleh ${user.name}.` : ''}
            </DialogDescription>
          </DialogHeader>

          {error && (
            <div className="mb-4">
              <Alert variant="error">{error}</Alert>
            </div>
          )}

          {outlets.length === 0 ? (
            <Alert variant="info">Belum ada outlet untuk ditugaskan.</Alert>
          ) : (
            <div className="space-y-2 rounded-control border border-border bg-surface p-3">
              {outlets.map((outlet) => (
                <label key={outlet.id} className="flex items-center gap-2 text-sm text-ink">
                  <input
                    type="checkbox"
                    checked={selected.includes(outlet.id)}
                    onChange={(event) => toggleOutlet(outlet.id, event.target.checked)}
                    className="h-4 w-4 rounded border-border text-accent-600 focus:ring-accent-500"
                  />
                  <Store className="h-3.5 w-3.5 text-muted" aria-hidden="true" />
                  {outlet.name}
                  <span className="text-xs text-muted">({outlet.code})</span>
                </label>
              ))}
            </div>
          )}
          {fieldErrors.outlet_ids && <p className="mt-1 text-xs text-danger">{fieldErrors.outlet_ids}</p>}
          <p className="mt-2 text-xs text-muted">
            Pengguna tanpa outlet tidak dapat membuka data operasional apa pun.
          </p>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={loading}>
              Batal
            </Button>
            <Button type="submit" loading={loading} disabled={outlets.length === 0}>
              Simpan Penugasan
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
