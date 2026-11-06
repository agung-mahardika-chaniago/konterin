import * as React from 'react';
import { Link } from 'react-router-dom';
import {
  Boxes,
  ChevronLeft,
  ChevronRight,
  Eye,
  Package,
  Pencil,
  Plus,
  Search,
  Settings2,
  Trash2,
} from 'lucide-react';

import { EmptyState } from '@/components/shared/empty-state';
import { Alert } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Select } from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { formatCurrency } from '@/features/dashboard/format';
import { hasPermission } from '@/lib/permissions';
import { useAuthStore } from '@/stores/auth-store';
import {
  deleteSparepart,
  getValuation,
  listCategories,
  listSpareparts,
  listStockAlerts,
  listSuppliers,
  listTopUsed,
} from '../api';
import { MasterDataDialog } from '../components/master-data-dialog';
import { SparepartFormDialog } from '../components/sparepart-form-dialog';
import type { Sparepart, SparepartCategory, StockAlert, Supplier, TopUsed, Valuation } from '../types';

export function SparepartListPage() {
  const user = useAuthStore((state) => state.user);
  const canCreate = hasPermission(user, 'sparepart.create');
  const canUpdate = hasPermission(user, 'sparepart.update');
  const canDelete = hasPermission(user, 'sparepart.delete');
  const canViewCost = hasPermission(user, 'inventory.cost.view');
  const canViewReport = hasPermission(user, 'report.view');

  const [items, setItems] = React.useState<Sparepart[]>([]);
  const [total, setTotal] = React.useState(0);
  const [page, setPage] = React.useState(1);
  const [totalPages, setTotalPages] = React.useState(1);
  const [search, setSearch] = React.useState('');
  const [debouncedSearch, setDebouncedSearch] = React.useState('');
  const [categoryId, setCategoryId] = React.useState('');
  const [lowStockOnly, setLowStockOnly] = React.useState(false);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);

  const [categories, setCategories] = React.useState<SparepartCategory[]>([]);
  const [suppliers, setSuppliers] = React.useState<Supplier[]>([]);
  const [alerts, setAlerts] = React.useState<StockAlert[]>([]);
  const [valuation, setValuation] = React.useState<Valuation | null>(null);
  const [topUsed, setTopUsed] = React.useState<TopUsed[]>([]);

  const [formOpen, setFormOpen] = React.useState(false);
  const [editing, setEditing] = React.useState<Sparepart | null>(null);
  const [masterOpen, setMasterOpen] = React.useState(false);

  React.useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search);
      setPage(1);
    }, 300);
    return () => clearTimeout(timer);
  }, [search]);

  const loadMasterData = React.useCallback(async () => {
    try {
      const [categoryList, supplierList] = await Promise.all([listCategories(), listSuppliers()]);
      setCategories(categoryList);
      setSuppliers(supplierList);
    } catch {
      // Master data is optional for rendering the list.
    }
  }, []);

  const loadData = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await listSpareparts({
        q: debouncedSearch.trim() || undefined,
        category_id: categoryId || undefined,
        low_stock: lowStockOnly || undefined,
        page,
        per_page: 15,
      });
      setItems(result.data ?? []);
      if (result.meta) {
        setTotal(result.meta.total);
        setTotalPages(result.meta.total_pages || 1);
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Gagal memuat daftar sparepart.');
    } finally {
      setLoading(false);
    }
  }, [debouncedSearch, categoryId, lowStockOnly, page]);

  const loadStats = React.useCallback(async () => {
    const tasks: Promise<void>[] = [
      listStockAlerts()
        .then((data) => setAlerts(data))
        .catch(() => setAlerts([])),
    ];
    if (canViewCost) {
      tasks.push(
        getValuation()
          .then((data) => setValuation(data))
          .catch(() => setValuation(null)),
      );
    }
    if (canViewReport) {
      tasks.push(
        listTopUsed()
          .then((data) => setTopUsed(data.slice(0, 5)))
          .catch(() => setTopUsed([])),
      );
    }
    await Promise.all(tasks);
  }, [canViewCost, canViewReport]);

  React.useEffect(() => {
    loadMasterData();
  }, [loadMasterData]);

  React.useEffect(() => {
    loadData();
  }, [loadData]);

  React.useEffect(() => {
    loadStats();
  }, [loadStats]);

  const refreshAll = () => {
    loadData();
    loadMasterData();
    loadStats();
  };

  const handleDelete = async (item: Sparepart) => {
    if (!window.confirm(`Yakin ingin menghapus sparepart "${item.name}"? Riwayat stok tetap disimpan.`)) {
      return;
    }
    try {
      await deleteSparepart(item.id);
      refreshAll();
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : 'Gagal menghapus sparepart.');
    }
  };

  const openCreate = () => {
    setEditing(null);
    setFormOpen(true);
  };

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-ink sm:text-[28px]">Sparepart</h1>
          <p className="mt-1 text-sm text-muted">Kelola master sparepart, stok, harga, dan peringatan stok minimum.</p>
        </div>
        <div className="flex flex-wrap gap-2 self-start sm:self-auto">
          {canCreate && (
            <Button variant="outline" onClick={() => setMasterOpen(true)} className="gap-2">
              <Settings2 className="h-4 w-4" aria-hidden="true" />
              Kategori & Supplier
            </Button>
          )}
          {canCreate && (
            <Button onClick={openCreate} className="gap-2">
              <Plus className="h-4 w-4" aria-hidden="true" />
              Sparepart Baru
            </Button>
          )}
        </div>
      </div>

      {error && <Alert variant="error">{error}</Alert>}

      {alerts.length > 0 && (
        <Alert variant="info">
          <p className="font-semibold">{alerts.length} sparepart di bawah stok minimum</p>
          <p className="mt-0.5 text-xs">
            {alerts.slice(0, 3).map((alert) => `${alert.name} (${alert.stock}/${alert.min_stock})`).join(', ')}
            {alerts.length > 3 ? `, +${alerts.length - 3} lainnya` : ''}
          </p>
        </Alert>
      )}

      <div className="grid gap-4 sm:grid-cols-3">
        <Card className="p-4">
          <p className="text-xs font-medium uppercase tracking-wide text-muted">Total Sparepart</p>
          <p className="mt-1 text-2xl font-bold text-ink">{total}</p>
        </Card>
        <Card className="p-4">
          <p className="text-xs font-medium uppercase tracking-wide text-muted">Stok Menipis</p>
          <p className="mt-1 text-2xl font-bold text-ink">{alerts.length}</p>
        </Card>
        {canViewCost && (
          <Card className="p-4">
            <p className="text-xs font-medium uppercase tracking-wide text-muted">Nilai Persediaan</p>
            <p className="mt-1 text-2xl font-bold text-ink">{formatCurrency(valuation?.total_value ?? 0)}</p>
            <p className="text-xs text-muted">{valuation?.total_stock ?? 0} unit</p>
          </Card>
        )}
      </div>

      <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
        <div className="relative flex-1 sm:max-w-md">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted" aria-hidden="true" />
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Cari nama, SKU, barcode, atau model..."
            className="pl-9"
          />
        </div>
        <Select
          value={categoryId}
          onChange={(e) => {
            setCategoryId(e.target.value);
            setPage(1);
          }}
          className="sm:max-w-[200px]"
        >
          <option value="">Semua kategori</option>
          {categories.map((category) => (
            <option key={category.id} value={category.id}>
              {category.name}
            </option>
          ))}
        </Select>
        <label className="flex items-center gap-2 whitespace-nowrap text-sm text-ink">
          <input
            type="checkbox"
            checked={lowStockOnly}
            onChange={(e) => {
              setLowStockOnly(e.target.checked);
              setPage(1);
            }}
            className="h-4 w-4 rounded border-border text-accent-600 focus:ring-accent-500"
          />
          Stok menipis
        </label>
      </div>

      <Card className="overflow-hidden">
        {loading ? (
          <div className="space-y-3 p-4">
            {[1, 2, 3, 4, 5].map((i) => (
              <div key={i} className="flex items-center justify-between py-2">
                <div className="space-y-1">
                  <Skeleton className="h-4 w-52" />
                  <Skeleton className="h-3 w-32" />
                </div>
                <Skeleton className="h-8 w-20" />
              </div>
            ))}
          </div>
        ) : items.length === 0 ? (
          <EmptyState
            icon={Boxes}
            title="Belum ada sparepart"
            description={
              search || categoryId || lowStockOnly
                ? 'Tidak ada sparepart yang sesuai dengan filter Anda.'
                : 'Tambahkan sparepart pertama agar teknisi dapat memakainya di tiket servis.'
            }
            action={
              canCreate ? (
                <Button onClick={openCreate} size="sm" className="gap-1.5">
                  <Plus className="h-4 w-4" aria-hidden="true" />
                  Tambah Sparepart
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
                    <th scope="col" className="px-5 py-3 font-semibold">Nama & SKU</th>
                    <th scope="col" className="px-5 py-3 font-semibold">Kategori</th>
                    <th scope="col" className="px-5 py-3 font-semibold text-center">Stok</th>
                    <th scope="col" className="px-5 py-3 font-semibold text-right">Harga Jual</th>
                    {canViewCost && <th scope="col" className="px-5 py-3 font-semibold text-right">Harga Modal</th>}
                    <th scope="col" className="px-5 py-3 font-semibold text-right">Aksi</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {items.map((item) => (
                    <tr key={item.id} className="transition-colors hover:bg-background/50">
                      <td className="px-5 py-3.5">
                        <div className="flex items-center gap-2">
                          <Link
                            to={`/spareparts/${item.id}`}
                            className="font-semibold text-ink transition-colors hover:text-accent-600"
                          >
                            {item.name}
                          </Link>
                          {!item.is_active && <Badge variant="neutral">Nonaktif</Badge>}
                        </div>
                        <p className="mt-0.5 text-xs text-muted">
                          {item.sku}
                          {item.barcode ? ` · ${item.barcode}` : ''}
                          {item.location ? ` · ${item.location}` : ''}
                        </p>
                      </td>
                      <td className="px-5 py-3.5 text-xs text-ink">{item.category_name || '—'}</td>
                      <td className="px-5 py-3.5 text-center">
                        <span className="font-semibold text-ink">{item.stock}</span>
                        {item.low_stock && (
                          <Badge variant="warning" className="ml-1.5">
                            Min {item.min_stock}
                          </Badge>
                        )}
                      </td>
                      <td className="px-5 py-3.5 text-right text-xs font-medium text-ink">
                        {formatCurrency(item.sell_price)}
                      </td>
                      {canViewCost && (
                        <td className="px-5 py-3.5 text-right text-xs text-muted">{formatCurrency(item.cost_price)}</td>
                      )}
                      <td className="whitespace-nowrap px-5 py-3.5 text-right">
                        <div className="flex items-center justify-end gap-1">
                          <Link to={`/spareparts/${item.id}`}>
                            <Button variant="ghost" size="sm" className="h-8 px-2" title="Lihat detail">
                              <Eye className="h-3.5 w-3.5 text-muted hover:text-ink" aria-hidden="true" />
                            </Button>
                          </Link>
                          {canUpdate && (
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => {
                                setEditing(item);
                                setFormOpen(true);
                              }}
                              className="h-8 px-2"
                              title="Ubah sparepart"
                            >
                              <Pencil className="h-3.5 w-3.5 text-muted hover:text-ink" aria-hidden="true" />
                            </Button>
                          )}
                          {canDelete && (
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => handleDelete(item)}
                              className="h-8 px-2 hover:text-danger"
                              title="Hapus sparepart"
                            >
                              <Trash2 className="h-3.5 w-3.5 text-muted hover:text-danger" aria-hidden="true" />
                            </Button>
                          )}
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            <div className="flex items-center justify-between border-t border-border px-5 py-3 text-xs text-muted">
              <span>
                Menampilkan {items.length} dari total {total} sparepart
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
                </Button>
              </div>
            </div>
          </>
        )}
      </Card>

      {canViewReport && topUsed.length > 0 && (
        <Card className="p-4">
          <div className="mb-3 flex items-center gap-2">
            <Package className="h-4 w-4 text-accent-600" aria-hidden="true" />
            <h2 className="text-sm font-semibold text-ink">Paling Banyak Dipakai (30 hari)</h2>
          </div>
          <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-5">
            {topUsed.map((part, index) => (
              <Link
                key={part.sparepart_id}
                to={`/spareparts/${part.sparepart_id}`}
                className="rounded-control border border-border p-3 transition-colors hover:border-accent-300"
              >
                <p className="text-xs text-muted">#{index + 1} · {part.sku}</p>
                <p className="truncate text-sm font-medium text-ink">{part.name}</p>
                <p className="mt-1 text-xs text-accent-600">{part.used_qty} unit dipakai</p>
              </Link>
            ))}
          </div>
        </Card>
      )}

      <SparepartFormDialog
        open={formOpen}
        onOpenChange={setFormOpen}
        sparepart={editing}
        categories={categories}
        suppliers={suppliers}
        onSuccess={refreshAll}
      />
      <MasterDataDialog
        open={masterOpen}
        onOpenChange={setMasterOpen}
        categories={categories}
        suppliers={suppliers}
        canManage={canCreate}
        onChanged={loadMasterData}
      />
    </div>
  );
}
