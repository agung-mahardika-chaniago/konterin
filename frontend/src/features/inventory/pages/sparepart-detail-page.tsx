import * as React from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import {
  ArrowLeft,
  Boxes,
  History,
  MapPin,
  Package,
  Pencil,
  Plus,
  ShoppingCart,
  SlidersHorizontal,
  Trash2,
} from 'lucide-react';

import { Alert } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { formatCurrency } from '@/features/dashboard/format';
import { hasPermission } from '@/lib/permissions';
import { useAuthStore } from '@/stores/auth-store';
import {
  deleteSparepart,
  getSparepart,
  listCategories,
  listMovements,
  listSuppliers,
} from '../api';
import { AdjustmentDialog } from '../components/adjustment-dialog';
import { SparepartFormDialog } from '../components/sparepart-form-dialog';
import { formatDateTime, movementTypeLabel, movementTypeVariant } from '../format';
import type { Movement, Sparepart, SparepartCategory, Supplier } from '../types';

export function SparepartDetailPage() {
  const { id = '' } = useParams();
  const navigate = useNavigate();
  const user = useAuthStore((state) => state.user);
  const canUpdate = hasPermission(user, 'sparepart.update');
  const canDelete = hasPermission(user, 'sparepart.delete');
  const canPurchase = hasPermission(user, 'inventory.purchase');
  const canAdjust = hasPermission(user, 'inventory.adjust');
  const canViewCost = hasPermission(user, 'inventory.cost.view');

  const [sparepart, setSparepart] = React.useState<Sparepart | null>(null);
  const [movements, setMovements] = React.useState<Movement[]>([]);
  const [categories, setCategories] = React.useState<SparepartCategory[]>([]);
  const [suppliers, setSuppliers] = React.useState<Supplier[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);
  const [editOpen, setEditOpen] = React.useState(false);
  const [adjustOpen, setAdjustOpen] = React.useState(false);

  const load = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [part, movementResult] = await Promise.all([
        getSparepart(id),
        listMovements({ sparepart_id: id, per_page: 20 }),
      ]);
      setSparepart(part);
      setMovements(movementResult.data ?? []);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Gagal memuat detail sparepart.');
    } finally {
      setLoading(false);
    }
  }, [id]);

  React.useEffect(() => {
    load();
  }, [load]);

  React.useEffect(() => {
    Promise.all([listCategories(), listSuppliers()])
      .then(([categoryList, supplierList]) => {
        setCategories(categoryList);
        setSuppliers(supplierList);
      })
      .catch(() => undefined);
  }, []);

  const handleDelete = async () => {
    if (!sparepart) return;
    if (!window.confirm(`Yakin ingin menghapus sparepart "${sparepart.name}"? Riwayat stok tetap disimpan.`)) {
      return;
    }
    try {
      await deleteSparepart(sparepart.id);
      navigate('/spareparts');
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : 'Gagal menghapus sparepart.');
    }
  };

  if (loading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-52" />
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {[1, 2, 3, 4].map((i) => (
            <Skeleton key={i} className="h-24" />
          ))}
        </div>
        <Skeleton className="h-64" />
      </div>
    );
  }

  if (error || !sparepart) {
    return (
      <div className="space-y-4">
        <Alert variant="error">{error ?? 'Sparepart tidak ditemukan.'}</Alert>
        <Link to="/spareparts">
          <Button variant="outline" className="gap-2">
            <ArrowLeft className="h-4 w-4" aria-hidden="true" />
            Kembali ke daftar
          </Button>
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div className="flex items-start gap-3">
          <Link to="/spareparts">
            <Button variant="ghost" size="icon" title="Kembali">
              <ArrowLeft className="h-4 w-4" aria-hidden="true" />
            </Button>
          </Link>
          <div>
            <div className="flex flex-wrap items-center gap-2">
              <h1 className="text-2xl font-bold tracking-tight text-ink">{sparepart.name}</h1>
              {sparepart.is_active ? (
                <Badge variant="success">Aktif</Badge>
              ) : (
                <Badge variant="neutral">Nonaktif</Badge>
              )}
              {sparepart.low_stock && <Badge variant="warning">Stok Menipis</Badge>}
            </div>
            <p className="mt-1 text-sm text-muted">
              {sparepart.sku}
              {sparepart.barcode ? ` · Barcode ${sparepart.barcode}` : ''}
              {sparepart.category_name ? ` · ${sparepart.category_name}` : ''}
            </p>
          </div>
        </div>
        <div className="flex flex-wrap gap-2">
          {canPurchase && (
            <Link to={`/inventory/purchases/new?sparepart=${sparepart.id}`}>
              <Button variant="outline" className="gap-2">
                <ShoppingCart className="h-4 w-4" aria-hidden="true" />
                Beli Stok
              </Button>
            </Link>
          )}
          {canAdjust && (
            <Button variant="outline" onClick={() => setAdjustOpen(true)} className="gap-2">
              <SlidersHorizontal className="h-4 w-4" aria-hidden="true" />
              Penyesuaian
            </Button>
          )}
          {canUpdate && (
            <Button variant="outline" onClick={() => setEditOpen(true)} className="gap-2">
              <Pencil className="h-4 w-4" aria-hidden="true" />
              Ubah
            </Button>
          )}
          {canDelete && (
            <Button variant="ghost" onClick={handleDelete} className="gap-2 text-danger hover:bg-red-50">
              <Trash2 className="h-4 w-4" aria-hidden="true" />
              Hapus
            </Button>
          )}
        </div>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Card className="p-4">
          <div className="flex items-center gap-2 text-muted">
            <Package className="h-4 w-4" aria-hidden="true" />
            <p className="text-xs font-medium uppercase tracking-wide">Stok Saat Ini</p>
          </div>
          <p className="mt-2 text-2xl font-bold text-ink">{sparepart.stock}</p>
          <p className="text-xs text-muted">Minimum {sparepart.min_stock} unit</p>
        </Card>
        <Card className="p-4">
          <p className="text-xs font-medium uppercase tracking-wide text-muted">Harga Jual</p>
          <p className="mt-2 text-2xl font-bold text-ink">{formatCurrency(sparepart.sell_price)}</p>
        </Card>
        {canViewCost && (
          <>
            <Card className="p-4">
              <p className="text-xs font-medium uppercase tracking-wide text-muted">Harga Modal</p>
              <p className="mt-2 text-2xl font-bold text-ink">{formatCurrency(sparepart.cost_price)}</p>
            </Card>
            <Card className="p-4">
              <p className="text-xs font-medium uppercase tracking-wide text-muted">Nilai Stok</p>
              <p className="mt-2 text-2xl font-bold text-ink">{formatCurrency(sparepart.stock_value)}</p>
            </Card>
          </>
        )}
      </div>

      <div className="grid gap-4 lg:grid-cols-3">
        <Card className="lg:col-span-1">
          <CardHeader>
            <CardTitle className="text-base">Informasi Master</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-sm">
            <div className="flex justify-between gap-3">
              <span className="text-muted">Supplier</span>
              <span className="text-right font-medium text-ink">{sparepart.supplier_name || '—'}</span>
            </div>
            <div className="flex justify-between gap-3">
              <span className="text-muted">Model kompatibel</span>
              <span className="max-w-[60%] text-right text-ink">{sparepart.compatible_models || '—'}</span>
            </div>
            <div className="flex justify-between gap-3">
              <span className="flex items-center gap-1 text-muted">
                <MapPin className="h-3.5 w-3.5" aria-hidden="true" />
                Lokasi
              </span>
              <span className="text-right font-medium text-ink">{sparepart.location || '—'}</span>
            </div>
            <div className="flex justify-between gap-3">
              <span className="text-muted">Stok terakhir</span>
              <span className="text-right text-ink">{formatDateTime(sparepart.last_stock_at)}</span>
            </div>
            <div className="flex justify-between gap-3">
              <span className="text-muted">Dibuat</span>
              <span className="text-right text-ink">{formatDateTime(sparepart.created_at)}</span>
            </div>
          </CardContent>
        </Card>

        <Card className="lg:col-span-2">
          <CardHeader className="flex flex-row items-center justify-between gap-3">
            <div className="flex items-center gap-2">
              <History className="h-4 w-4 text-accent-600" aria-hidden="true" />
              <CardTitle className="text-base">Riwayat Stok Terbaru</CardTitle>
            </div>
            <Link to={`/inventory/movements?sparepart_id=${sparepart.id}`}>
              <Button variant="ghost" size="sm" className="text-xs">
                Lihat semua
              </Button>
            </Link>
          </CardHeader>
          <CardContent>
            {movements.length === 0 ? (
              <div className="flex flex-col items-center py-8 text-center">
                <Boxes className="mb-2 h-8 w-8 text-muted" aria-hidden="true" />
                <p className="text-sm text-muted">Belum ada pergerakan stok.</p>
                {canPurchase && (
                  <Link to={`/inventory/purchases/new?sparepart=${sparepart.id}`} className="mt-3">
                    <Button size="sm" className="gap-1.5">
                      <Plus className="h-4 w-4" aria-hidden="true" />
                      Catat Pembelian
                    </Button>
                  </Link>
                )}
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs">
                  <thead className="border-b border-border text-[11px] uppercase tracking-wide text-muted">
                    <tr>
                      <th className="py-2 pr-3 font-semibold">Waktu</th>
                      <th className="py-2 pr-3 font-semibold">Tipe</th>
                      <th className="py-2 pr-3 text-right font-semibold">Qty</th>
                      <th className="py-2 pr-3 text-right font-semibold">Stok</th>
                      {canViewCost && <th className="py-2 pr-3 text-right font-semibold">Modal</th>}
                      <th className="py-2 font-semibold">Catatan</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-border">
                    {movements.map((movement) => (
                      <tr key={movement.id}>
                        <td className="py-2.5 pr-3 text-muted">{formatDateTime(movement.created_at)}</td>
                        <td className="py-2.5 pr-3">
                          <Badge variant={movementTypeVariant(movement.movement_type)}>
                            {movementTypeLabel(movement.movement_type)}
                          </Badge>
                        </td>
                        <td
                          className={`py-2.5 pr-3 text-right font-semibold ${
                            movement.qty > 0 ? 'text-emerald-600' : 'text-red-600'
                          }`}
                        >
                          {movement.qty > 0 ? `+${movement.qty}` : movement.qty}
                        </td>
                        <td className="py-2.5 pr-3 text-right text-ink">
                          {movement.stock_before} → {movement.stock_after}
                        </td>
                        {canViewCost && (
                          <td className="py-2.5 pr-3 text-right text-muted">{formatCurrency(movement.unit_cost)}</td>
                        )}
                        <td className="max-w-[220px] truncate py-2.5 text-muted" title={movement.notes ?? ''}>
                          {movement.notes || movement.reference_type}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      <SparepartFormDialog
        open={editOpen}
        onOpenChange={setEditOpen}
        sparepart={sparepart}
        categories={categories}
        suppliers={suppliers}
        onSuccess={(saved) => setSparepart(saved)}
      />
      <AdjustmentDialog
        open={adjustOpen}
        onOpenChange={setAdjustOpen}
        sparepart={sparepart}
        onSuccess={load}
      />
    </div>
  );
}
