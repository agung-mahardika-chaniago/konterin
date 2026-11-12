import * as React from 'react';
import { MapPin, Pencil, Phone, Plus, Store } from 'lucide-react';

import { EmptyState } from '@/components/shared/empty-state';
import { Alert } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { formatDateTime } from '@/features/invoices/format';
import { hasPermission } from '@/lib/permissions';
import { useAuthStore } from '@/stores/auth-store';
import { listOutlets } from '../api';
import { OutletFormDialog } from '../components/outlet-form-dialog';
import type { Outlet } from '../types';

export function OutletListPage() {
  const user = useAuthStore((state) => state.user);
  const canManage = hasPermission(user, 'outlet.manage');

  const [outlets, setOutlets] = React.useState<Outlet[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);
  const [success, setSuccess] = React.useState<string | null>(null);
  const [formOpen, setFormOpen] = React.useState(false);
  const [editing, setEditing] = React.useState<Outlet | null>(null);

  const load = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const items = await listOutlets();
      setOutlets(items ?? []);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Gagal memuat daftar outlet.');
    } finally {
      setLoading(false);
    }
  }, []);

  React.useEffect(() => {
    load();
  }, [load]);

  const handleSaved = (saved: Outlet, created: boolean) => {
    setSuccess(`Outlet ${saved.name} berhasil ${created ? 'ditambahkan' : 'diperbarui'}.`);
    if (created) {
      setOutlets((current) => [...current, saved]);
    } else {
      setOutlets((current) => current.map((item) => (item.id === saved.id ? saved : item)));
    }
  };

  const openCreate = () => {
    setEditing(null);
    setFormOpen(true);
  };

  const openEdit = (outlet: Outlet) => {
    setEditing(outlet);
    setFormOpen(true);
  };

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-ink sm:text-[28px]">Outlet</h1>
          <p className="mt-1 text-sm text-muted">
            Kelola cabang toko — setiap outlet punya stok, nomor dokumen, dan laporan sendiri.
          </p>
        </div>
        {canManage && (
          <Button onClick={openCreate} className="gap-2 self-start sm:self-auto">
            <Plus className="h-4 w-4" aria-hidden="true" />
            Tambah Outlet
          </Button>
        )}
      </div>

      {success && <Alert variant="success">{success}</Alert>}
      {error && <Alert variant="error">{error}</Alert>}

      <Card className="overflow-hidden">
        {loading ? (
          <div className="space-y-3 p-4">
            {[1, 2, 3].map((i) => (
              <div key={i} className="flex items-center justify-between py-2">
                <div className="space-y-1">
                  <Skeleton className="h-4 w-48" />
                  <Skeleton className="h-3 w-32" />
                </div>
                <Skeleton className="h-8 w-24" />
              </div>
            ))}
          </div>
        ) : outlets.length === 0 ? (
          <EmptyState
            icon={Store}
            title="Belum ada outlet"
            description="Tambahkan outlet pertama untuk mulai mengoperasikan toko."
            action={
              canManage ? (
                <Button onClick={openCreate} size="sm" className="gap-1.5">
                  <Plus className="h-4 w-4" aria-hidden="true" />
                  Tambah Outlet
                </Button>
              ) : undefined
            }
            className="py-12"
          />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead className="border-b border-border bg-background text-xs uppercase tracking-wider text-muted">
                <tr>
                  <th scope="col" className="px-5 py-3 font-semibold">Outlet</th>
                  <th scope="col" className="px-5 py-3 font-semibold">Alamat</th>
                  <th scope="col" className="px-5 py-3 font-semibold">Telepon</th>
                  <th scope="col" className="px-5 py-3 font-semibold">Zona Waktu</th>
                  <th scope="col" className="px-5 py-3 font-semibold">Status</th>
                  <th scope="col" className="px-5 py-3 font-semibold">Diperbarui</th>
                  {canManage && <th scope="col" className="px-5 py-3 text-right font-semibold">Aksi</th>}
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {outlets.map((outlet) => (
                  <tr key={outlet.id} className="transition-colors hover:bg-background/50">
                    <td className="px-5 py-3.5">
                      <p className="font-medium text-ink">{outlet.name}</p>
                      <p className="font-mono text-[11px] uppercase text-muted">{outlet.code}</p>
                    </td>
                    <td className="max-w-[280px] px-5 py-3.5 text-xs text-muted">
                      {outlet.address ? (
                        <span className="flex items-start gap-1.5">
                          <MapPin className="mt-0.5 h-3.5 w-3.5 shrink-0" aria-hidden="true" />
                          {outlet.address}
                        </span>
                      ) : (
                        '—'
                      )}
                    </td>
                    <td className="px-5 py-3.5 text-xs text-muted">
                      {outlet.phone ? (
                        <span className="flex items-center gap-1.5">
                          <Phone className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
                          {outlet.phone}
                        </span>
                      ) : (
                        '—'
                      )}
                    </td>
                    <td className="px-5 py-3.5 font-mono text-xs text-ink">{outlet.timezone}</td>
                    <td className="px-5 py-3.5">
                      {outlet.is_active ? <Badge variant="success">Aktif</Badge> : <Badge variant="neutral">Nonaktif</Badge>}
                    </td>
                    <td className="px-5 py-3.5 text-xs text-muted">{formatDateTime(outlet.updated_at)}</td>
                    {canManage && (
                      <td className="whitespace-nowrap px-5 py-3.5 text-right">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => openEdit(outlet)}
                          className="h-8 px-2"
                          title="Ubah outlet"
                        >
                          <Pencil className="h-3.5 w-3.5 text-muted hover:text-ink" aria-hidden="true" />
                        </Button>
                      </td>
                    )}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>

      <OutletFormDialog
        open={formOpen}
        onOpenChange={setFormOpen}
        outlet={editing}
        onSuccess={handleSaved}
      />
    </div>
  );
}
