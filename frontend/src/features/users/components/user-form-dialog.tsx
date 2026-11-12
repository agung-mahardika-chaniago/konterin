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
import { ApiError } from '@/lib/api-client';
import { createUser, updateUser } from '../api';
import type { RoleInfo, User } from '../types';

interface OutletOption {
  id: string;
  name: string;
  code: string;
}

interface UserFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  roles: RoleInfo[];
  outlets: OutletOption[];
  user?: User | null;
  onSuccess: (saved: User, created: boolean) => void;
}

export function UserFormDialog({ open, onOpenChange, roles, outlets, user, onSuccess }: UserFormDialogProps) {
  const isEdit = Boolean(user);

  const [name, setName] = React.useState('');
  const [email, setEmail] = React.useState('');
  const [phone, setPhone] = React.useState('');
  const [role, setRole] = React.useState('');
  const [password, setPassword] = React.useState('');
  const [isActive, setIsActive] = React.useState(true);
  const [outletIds, setOutletIds] = React.useState<string[]>([]);
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = React.useState<Record<string, string>>({});

  React.useEffect(() => {
    if (!open) return;
    setName(user?.name ?? '');
    setEmail(user?.email ?? '');
    setPhone(user?.phone ?? '');
    setRole(user?.role ?? roles[0]?.slug ?? '');
    setPassword('');
    setIsActive(user?.is_active ?? true);
    setOutletIds(user?.outlets.map((outlet) => outlet.id) ?? []);
    setError(null);
    setFieldErrors({});
    // Roles/outlets are intentionally excluded: reloading must not wipe input.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, user]);

  const toggleOutlet = (outletId: string, checked: boolean) => {
    setOutletIds((current) =>
      checked ? [...current, outletId] : current.filter((id) => id !== outletId),
    );
  };

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setLoading(true);
    setError(null);
    setFieldErrors({});
    try {
      let saved: User;
      if (user) {
        saved = await updateUser(user.id, {
          name: name.trim(),
          phone: phone.trim() || null,
          role,
          is_active: isActive,
        });
        onSuccess(saved, false);
      } else {
        saved = await createUser({
          name: name.trim(),
          email: email.trim(),
          phone: phone.trim() || null,
          role,
          password,
          outlet_ids: outletIds,
          is_active: isActive,
        });
        onSuccess(saved, true);
      }
      onOpenChange(false);
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        setError(err.message);
        setFieldErrors(err.fieldErrors());
      } else {
        setError('Gagal menyimpan pengguna.');
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
            <DialogTitle>{isEdit ? 'Ubah Pengguna' : 'Tambah Pengguna'}</DialogTitle>
            <DialogDescription>
              {isEdit
                ? 'Perubahan langsung berlaku pada sesi berikutnya pengguna.'
                : 'Pengguna baru wajib mengganti kata sandi saat login pertama.'}
            </DialogDescription>
          </DialogHeader>

          {error && (
            <div className="mb-4">
              <Alert variant="error">{error}</Alert>
            </div>
          )}

          <div className="space-y-4">
            <div className="space-y-1.5">
              <Label htmlFor="user-name">Nama *</Label>
              <Input
                id="user-name"
                value={name}
                onChange={(event) => setName(event.target.value)}
                maxLength={120}
                required
              />
              {fieldErrors.name && <p className="text-xs text-danger">{fieldErrors.name}</p>}
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="user-email">Email</Label>
              {isEdit ? (
                <>
                  <Input id="user-email" value={email} disabled />
                  <p className="text-xs text-muted">Email tidak dapat diubah setelah akun dibuat.</p>
                </>
              ) : (
                <>
                  <Input
                    id="user-email"
                    type="email"
                    value={email}
                    onChange={(event) => setEmail(event.target.value)}
                    maxLength={150}
                    required
                  />
                  {fieldErrors.email && <p className="text-xs text-danger">{fieldErrors.email}</p>}
                </>
              )}
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-1.5">
                <Label htmlFor="user-phone">Telepon</Label>
                <Input
                  id="user-phone"
                  value={phone}
                  onChange={(event) => setPhone(event.target.value)}
                  maxLength={30}
                  placeholder="08xxxxxxxxxx"
                />
                {fieldErrors.phone && <p className="text-xs text-danger">{fieldErrors.phone}</p>}
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="user-role">Peran *</Label>
                <Select id="user-role" value={role} onChange={(event) => setRole(event.target.value)} required>
                  {roles.map((item) => (
                    <option key={item.slug} value={item.slug}>
                      {item.name}
                    </option>
                  ))}
                </Select>
                {fieldErrors.role && <p className="text-xs text-danger">{fieldErrors.role}</p>}
              </div>
            </div>

            {!isEdit && (
              <div className="space-y-1.5">
                <Label htmlFor="user-password">Kata Sandi Awal *</Label>
                <Input
                  id="user-password"
                  type="password"
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  minLength={8}
                  maxLength={72}
                  required
                  autoComplete="new-password"
                />
                {fieldErrors.password && <p className="text-xs text-danger">{fieldErrors.password}</p>}
                <p className="text-xs text-muted">Minimal 8 karakter. Pengguna akan diminta menggantinya.</p>
              </div>
            )}

            {outlets.length > 0 && (
              <div className="space-y-1.5">
                <Label>Outlet yang Dapat Diakses</Label>
                <div className="space-y-2 rounded-control border border-border bg-surface p-3">
                  {outlets.map((outlet) => (
                    <label key={outlet.id} className="flex items-center gap-2 text-sm text-ink">
                      <input
                        type="checkbox"
                        checked={outletIds.includes(outlet.id)}
                        onChange={(event) => toggleOutlet(outlet.id, event.target.checked)}
                        className="h-4 w-4 rounded border-border text-accent-600 focus:ring-accent-500"
                      />
                      {outlet.name}
                      <span className="text-xs text-muted">({outlet.code})</span>
                    </label>
                  ))}
                </div>
                {!isEdit && fieldErrors.outlet_ids && (
                  <p className="text-xs text-danger">{fieldErrors.outlet_ids}</p>
                )}
              </div>
            )}

            <label htmlFor="user-active" className="flex items-start gap-3 rounded-control border border-border bg-surface p-3">
              <input
                id="user-active"
                type="checkbox"
                checked={isActive}
                onChange={(event) => setIsActive(event.target.checked)}
                className="mt-0.5 h-4 w-4 rounded border-border text-accent-600 focus:ring-accent-500"
              />
              <span>
                <span className="block text-sm font-medium text-ink">Akun aktif</span>
                <span className="block text-xs text-muted">Pengguna nonaktif tidak dapat masuk ke aplikasi.</span>
              </span>
            </label>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={loading}>
              Batal
            </Button>
            <Button type="submit" loading={loading}>
              {isEdit ? 'Simpan Perubahan' : 'Tambah Pengguna'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
