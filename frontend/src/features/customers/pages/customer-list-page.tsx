import * as React from 'react';
import { Link } from 'react-router-dom';
import {
  ChevronLeft,
  ChevronRight,
  Eye,
  Pencil,
  Plus,
  Search,
  Trash2,
  Users,
} from 'lucide-react';

import { EmptyState } from '@/components/shared/empty-state';
import { Alert } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Skeleton } from '@/components/ui/skeleton';
import { deleteCustomer, listCustomers } from '../api';
import { CustomerFormDialog } from '../components/customer-form-dialog';
import type { Customer } from '../types';

export function CustomerListPage() {
  const [customers, setCustomers] = React.useState<Customer[]>([]);
  const [total, setTotal] = React.useState(0);
  const [page, setPage] = React.useState(1);
  const [totalPages, setTotalPages] = React.useState(1);
  const [search, setSearch] = React.useState('');
  const [debouncedSearch, setDebouncedSearch] = React.useState('');
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);

  // Modal state
  const [dialogOpen, setDialogOpen] = React.useState(false);
  const [editingCustomer, setEditingCustomer] = React.useState<Customer | null>(null);

  React.useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search);
      setPage(1);
    }, 300);
    return () => clearTimeout(timer);
  }, [search]);

  const loadData = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await listCustomers({
        q: debouncedSearch.trim() || undefined,
        page,
        per_page: 15,
      });
      setCustomers(res.data ?? []);
      if (res.meta) {
        setTotal(res.meta.total);
        setTotalPages(res.meta.total_pages || 1);
      }
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Gagal memuat daftar pelanggan.';
      setError(message);
    } finally {
      setLoading(false);
    }
  }, [debouncedSearch, page]);

  React.useEffect(() => {
    loadData();
  }, [loadData]);

  const handleDelete = async (customer: Customer) => {
    if (!window.confirm(`Yakin ingin menghapus pelanggan "${customer.name}"?`)) {
      return;
    }
    try {
      await deleteCustomer(customer.id);
      loadData();
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : 'Gagal menghapus pelanggan.');
    }
  };

  const handleCreateNew = () => {
    setEditingCustomer(null);
    setDialogOpen(true);
  };

  const handleEdit = (customer: Customer) => {
    setEditingCustomer(customer);
    setDialogOpen(true);
  };

  const formatLastService = (dateStr?: string | null) => {
    if (!dateStr) return '-';
    return new Date(dateStr).toLocaleDateString('id-ID', {
      day: 'numeric',
      month: 'short',
      year: 'numeric',
    });
  };

  return (
    <div className="space-y-5">
      {/* Page Header */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-ink sm:text-[28px]">Pelanggan</h1>
          <p className="mt-1 text-sm text-muted">
            Kelola data kontak, riwayat servis, dan perangkat milik pelanggan.
          </p>
        </div>
        <Button onClick={handleCreateNew} className="gap-2 self-start sm:self-auto">
          <Plus className="h-4 w-4" aria-hidden="true" />
          Pelanggan Baru
        </Button>
      </div>

      {error && <Alert variant="error">{error}</Alert>}

      {/* Filter / Search Bar */}
      <div className="flex items-center gap-3">
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted" aria-hidden="true" />
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Cari berdasarkan nama atau no. telepon..."
            className="pl-9"
          />
        </div>
      </div>

      {/* Table Card */}
      <Card className="overflow-hidden">
        {loading ? (
          <div className="divide-y divide-border p-4 space-y-3">
            {[1, 2, 3, 4, 5].map((i) => (
              <div key={i} className="flex items-center justify-between py-2">
                <div className="space-y-1">
                  <Skeleton className="h-4 w-48" />
                  <Skeleton className="h-3 w-32" />
                </div>
                <Skeleton className="h-8 w-20" />
              </div>
            ))}
          </div>
        ) : customers.length === 0 ? (
          <EmptyState
            icon={Users}
            title="Belum ada data pelanggan"
            description={
              search
                ? `Tidak ada pelanggan yang sesuai dengan pencarian "${search}".`
                : 'Mulai dengan menambahkan data pelanggan pertama Anda.'
            }
            action={
              <Button onClick={handleCreateNew} size="sm" className="gap-1.5">
                <Plus className="h-4 w-4" aria-hidden="true" />
                Tambah Pelanggan
              </Button>
            }
            className="py-12"
          />
        ) : (
          <>
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead className="bg-background text-xs uppercase tracking-wider text-muted border-b border-border">
                  <tr>
                    <th scope="col" className="px-5 py-3 font-semibold">Nama & Kontak</th>
                    <th scope="col" className="px-5 py-3 font-semibold">Alamat</th>
                    <th scope="col" className="px-5 py-3 font-semibold text-center">Servis Terakhir</th>
                    <th scope="col" className="px-5 py-3 font-semibold text-right">Aksi</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {customers.map((c) => (
                    <tr key={c.id} className="hover:bg-background/50 transition-colors">
                      <td className="px-5 py-3.5">
                        <Link
                          to={`/customers/${c.id}`}
                          className="font-semibold text-ink hover:text-accent-600 transition-colors"
                        >
                          {c.name}
                        </Link>
                        <p className="text-xs text-muted mt-0.5">{c.phone}</p>
                      </td>
                      <td className="px-5 py-3.5 text-xs text-ink max-w-xs truncate">
                        {c.address || <span className="text-muted italic">Tidak ada alamat</span>}
                      </td>
                      <td className="px-5 py-3.5 text-xs text-center text-muted">
                        {formatLastService(c.last_service_at)}
                      </td>
                      <td className="px-5 py-3.5 text-right whitespace-nowrap">
                        <div className="flex items-center justify-end gap-1">
                          <Link to={`/customers/${c.id}`}>
                            <Button variant="ghost" size="sm" className="h-8 px-2 text-xs" title="Lihat detail">
                              <Eye className="h-3.5 w-3.5 text-muted hover:text-ink" aria-hidden="true" />
                              <span className="sr-only">Lihat detail</span>
                            </Button>
                          </Link>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleEdit(c)}
                            className="h-8 px-2 text-xs"
                            title="Ubah data"
                          >
                            <Pencil className="h-3.5 w-3.5 text-muted hover:text-ink" aria-hidden="true" />
                            <span className="sr-only">Ubah</span>
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleDelete(c)}
                            className="h-8 px-2 text-xs hover:text-danger"
                            title="Hapus pelanggan"
                          >
                            <Trash2 className="h-3.5 w-3.5 text-muted hover:text-danger" aria-hidden="true" />
                            <span className="sr-only">Hapus</span>
                          </Button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {/* Pagination Controls */}
            <div className="flex items-center justify-between border-t border-border px-5 py-3 text-xs text-muted">
              <span>
                Menampilkan {customers.length} dari total {total} pelanggan
              </span>
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  disabled={page <= 1}
                  className="h-8 px-2"
                >
                  <ChevronLeft className="h-4 w-4" aria-hidden="true" />
                  <span className="sr-only">Sebelumnya</span>
                </Button>
                <span className="font-medium text-ink">
                  {page} / {totalPages}
                </span>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                  disabled={page >= totalPages}
                  className="h-8 px-2"
                >
                  <ChevronRight className="h-4 w-4" aria-hidden="true" />
                  <span className="sr-only">Berikutnya</span>
                </Button>
              </div>
            </div>
          </>
        )}
      </Card>

      {/* Customer Form Modal */}
      <CustomerFormDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        customer={editingCustomer}
        onSuccess={() => loadData()}
      />
    </div>
  );
}
