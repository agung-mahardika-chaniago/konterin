import * as React from 'react';
import {
  ChevronLeft,
  ChevronRight,
  KeyRound,
  Pencil,
  Plus,
  Search,
  Store,
  Trash2,
  Users as UsersIcon,
} from 'lucide-react';

import { EmptyState } from '@/components/shared/empty-state';
import { Alert } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Select } from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { listOutlets } from '@/features/outlets/api';
import type { Outlet } from '@/features/outlets/types';
import { formatDateTime } from '@/features/invoices/format';
import { hasPermission } from '@/lib/permissions';
import { useAuthStore } from '@/stores/auth-store';
import { deleteUser, listRoles, listUsers } from '../api';
import { ResetPasswordDialog } from '../components/reset-password-dialog';
import { UserFormDialog } from '../components/user-form-dialog';
import { UserOutletsDialog } from '../components/user-outlets-dialog';
import type { RoleInfo, User } from '../types';

export function UserListPage() {
  const user = useAuthStore((state) => state.user);
  const canManage = hasPermission(user, 'user.manage');

  const [items, setItems] = React.useState<User[]>([]);
  const [total, setTotal] = React.useState(0);
  const [page, setPage] = React.useState(1);
  const [totalPages, setTotalPages] = React.useState(1);
  const [search, setSearch] = React.useState('');
  const [debouncedSearch, setDebouncedSearch] = React.useState('');
  const [roleFilter, setRoleFilter] = React.useState('');
  const [statusFilter, setStatusFilter] = React.useState('');
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);
  const [success, setSuccess] = React.useState<string | null>(null);

  const [roles, setRoles] = React.useState<RoleInfo[]>([]);
  const [outlets, setOutlets] = React.useState<Outlet[]>([]);

  const [formOpen, setFormOpen] = React.useState(false);
  const [editing, setEditing] = React.useState<User | null>(null);
  const [resetTarget, setResetTarget] = React.useState<User | null>(null);
  const [outletsTarget, setOutletsTarget] = React.useState<User | null>(null);

  React.useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search);
      setPage(1);
    }, 300);
    return () => clearTimeout(timer);
  }, [search]);

  const loadReference = React.useCallback(async () => {
    try {
      const [roleList, outletList] = await Promise.all([listRoles(), listOutlets()]);
      setRoles(roleList ?? []);
      setOutlets(outletList ?? []);
    } catch {
      setRoles([]);
      setOutlets([]);
    }
  }, []);

  const loadData = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await listUsers({
        q: debouncedSearch.trim() || undefined,
        role: roleFilter || undefined,
        status: statusFilter || undefined,
        page,
        per_page: 15,
      });
      setItems(result.data?.items ?? []);
      if (result.meta) {
        setTotal(result.meta.total);
        setTotalPages(result.meta.total_pages || 1);
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Gagal memuat daftar pengguna.');
    } finally {
      setLoading(false);
    }
  }, [debouncedSearch, roleFilter, statusFilter, page]);

  React.useEffect(() => {
    loadReference();
  }, [loadReference]);

  React.useEffect(() => {
    loadData();
  }, [loadData]);

  const handleSaved = (saved: User, created: boolean) => {
    setSuccess(`Pengguna ${saved.name} berhasil ${created ? 'ditambahkan' : 'diperbarui'}.`);
    loadData();
  };

  const handleOutletsSaved = (saved: User) => {
    setItems((current) => current.map((item) => (item.id === saved.id ? saved : item)));
    setSuccess(`Penugasan outlet untuk ${saved.name} berhasil disimpan.`);
  };

  const handleDelete = async (target: User) => {
    if (!window.confirm(`Hapus pengguna "${target.name}" (${target.email})? Mereka tidak dapat masuk lagi.`)) {
      return;
    }
    try {
      await deleteUser(target.id);
      setSuccess(`Pengguna ${target.name} berhasil dihapus.`);
      loadData();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Gagal menghapus pengguna.');
    }
  };

  const openCreate = () => {
    setEditing(null);
    setFormOpen(true);
  };

  const openEdit = (target: User) => {
    setEditing(target);
    setFormOpen(true);
  };

  const resetFilters = () => {
    setSearch('');
    setRoleFilter('');
    setStatusFilter('');
    setPage(1);
  };

  const hasFilter = Boolean(debouncedSearch || roleFilter || statusFilter);

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-ink sm:text-[28px]">Pengguna</h1>
          <p className="mt-1 text-sm text-muted">
            Kelola akun staf, peran, dan outlet yang dapat mereka akses.
          </p>
        </div>
        {canManage && (
          <Button onClick={openCreate} className="gap-2 self-start sm:self-auto">
            <Plus className="h-4 w-4" aria-hidden="true" />
            Tambah Pengguna
          </Button>
        )}
      </div>

      {success && <Alert variant="success">{success}</Alert>}
      {error && <Alert variant="error">{error}</Alert>}

      <div className="flex flex-col gap-3 lg:flex-row lg:items-center">
        <div className="relative flex-1 lg:max-w-md">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted" aria-hidden="true" />
          <Input
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder="Cari nama, email, atau telepon..."
            className="pl-9"
          />
        </div>
        <Select
          value={roleFilter}
          onChange={(event) => {
            setRoleFilter(event.target.value);
            setPage(1);
          }}
          className="lg:max-w-[180px]"
        >
          <option value="">Semua peran</option>
          {roles.map((role) => (
            <option key={role.slug} value={role.slug}>
              {role.name}
            </option>
          ))}
        </Select>
        <Select
          value={statusFilter}
          onChange={(event) => {
            setStatusFilter(event.target.value);
            setPage(1);
          }}
          className="lg:max-w-[150px]"
        >
          <option value="">Semua status</option>
          <option value="active">Aktif</option>
          <option value="inactive">Nonaktif</option>
        </Select>
        {hasFilter && (
          <Button variant="ghost" size="sm" onClick={resetFilters}>
            Reset filter
          </Button>
        )}
      </div>

      <Card className="overflow-hidden">
        {loading ? (
          <div className="space-y-3 p-4">
            {[1, 2, 3, 4, 5].map((i) => (
              <div key={i} className="flex items-center justify-between py-2">
                <div className="space-y-1">
                  <Skeleton className="h-4 w-48" />
                  <Skeleton className="h-3 w-32" />
                </div>
                <Skeleton className="h-8 w-24" />
              </div>
            ))}
          </div>
        ) : items.length === 0 ? (
          <EmptyState
            icon={UsersIcon}
            title="Belum ada pengguna"
            description={
              hasFilter
                ? 'Tidak ada pengguna yang sesuai dengan filter Anda.'
                : 'Tambahkan staf pertama untuk mulai beroperasi.'
            }
            action={
              canManage ? (
                <Button onClick={openCreate} size="sm" className="gap-1.5">
                  <Plus className="h-4 w-4" aria-hidden="true" />
                  Tambah Pengguna
                </Button>
              ) : undefined
            }
            className="py-12"
          />
        ) : (
          <>
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead className="border-b border-border bg-background text-xs uppercase tracking-wider text-muted">
                  <tr>
                    <th scope="col" className="px-5 py-3 font-semibold">Pengguna</th>
                    <th scope="col" className="px-5 py-3 font-semibold">Peran</th>
                    <th scope="col" className="px-5 py-3 font-semibold">Outlet</th>
                    <th scope="col" className="px-5 py-3 font-semibold">Status</th>
                    <th scope="col" className="px-5 py-3 font-semibold">Login Terakhir</th>
                    {canManage && <th scope="col" className="px-5 py-3 text-right font-semibold">Aksi</th>}
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {items.map((item) => (
                    <tr key={item.id} className="transition-colors hover:bg-background/50">
                      <td className="px-5 py-3.5">
                        <p className="font-medium text-ink">{item.name}</p>
                        <p className="text-xs text-muted">{item.email}</p>
                        {item.phone && <p className="text-[11px] text-muted">{item.phone}</p>}
                      </td>
                      <td className="px-5 py-3.5">
                        <Badge variant={item.role === 'owner' ? 'brand' : 'neutral'}>{item.role_name}</Badge>
                      </td>
                      <td className="px-5 py-3.5 text-xs text-muted">
                        {item.outlets.length === 0 ? (
                          <span className="text-amber-600">Tanpa outlet</span>
                        ) : (
                          <span>
                            {item.outlets
                              .slice(0, 2)
                              .map((outlet) => outlet.name)
                              .join(', ')}
                            {item.outlets.length > 2 ? ` +${item.outlets.length - 2} lainnya` : ''}
                          </span>
                        )}
                      </td>
                      <td className="px-5 py-3.5">
                        <div className="flex flex-col gap-1">
                          {item.is_active ? (
                            <Badge variant="success">Aktif</Badge>
                          ) : (
                            <Badge variant="danger">Nonaktif</Badge>
                          )}
                          {item.must_change_password && <Badge variant="warning">Perlu ganti sandi</Badge>}
                        </div>
                      </td>
                      <td className="px-5 py-3.5 text-xs text-muted">{formatDateTime(item.last_login_at)}</td>
                      {canManage && (
                        <td className="whitespace-nowrap px-5 py-3.5 text-right">
                          <div className="flex items-center justify-end gap-1">
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => openEdit(item)}
                              className="h-8 px-2"
                              title="Ubah pengguna"
                            >
                              <Pencil className="h-3.5 w-3.5 text-muted hover:text-ink" aria-hidden="true" />
                            </Button>
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => setOutletsTarget(item)}
                              className="h-8 px-2"
                              title="Atur outlet"
                            >
                              <Store className="h-3.5 w-3.5 text-muted hover:text-ink" aria-hidden="true" />
                            </Button>
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => setResetTarget(item)}
                              className="h-8 px-2"
                              title="Reset kata sandi"
                            >
                              <KeyRound className="h-3.5 w-3.5 text-muted hover:text-ink" aria-hidden="true" />
                            </Button>
                            {item.id !== user?.id && (
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => handleDelete(item)}
                                className="h-8 px-2 hover:text-danger"
                                title="Hapus pengguna"
                              >
                                <Trash2 className="h-3.5 w-3.5 text-muted hover:text-danger" aria-hidden="true" />
                              </Button>
                            )}
                          </div>
                        </td>
                      )}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            <div className="flex items-center justify-between border-t border-border px-5 py-3 text-xs text-muted">
              <span>
                Menampilkan {items.length} dari total {total} pengguna
              </span>
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage((current) => Math.max(1, current - 1))}
                  disabled={page <= 1}
                  className="h-8 px-2"
                >
                  <ChevronLeft className="h-4 w-4" aria-hidden="true" />
                </Button>
                <span className="font-medium text-ink">
                  {page} / {totalPages}
                </span>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage((current) => Math.min(totalPages, current + 1))}
                  disabled={page >= totalPages}
                  className="h-8 px-2"
                >
                  <ChevronRight className="h-4 w-4" aria-hidden="true" />
                </Button>
              </div>
            </div>
          </>
        )}
      </Card>

      <UserFormDialog
        open={formOpen}
        onOpenChange={setFormOpen}
        roles={roles}
        outlets={outlets}
        user={editing}
        onSuccess={handleSaved}
      />

      <UserOutletsDialog
        open={outletsTarget !== null}
        onOpenChange={(open) => {
          if (!open) setOutletsTarget(null);
        }}
        outlets={outlets}
        user={outletsTarget}
        onSuccess={handleOutletsSaved}
      />

      <ResetPasswordDialog
        open={resetTarget !== null}
        onOpenChange={(open) => {
          if (!open) setResetTarget(null);
        }}
        user={resetTarget}
      />
    </div>
  );
}
