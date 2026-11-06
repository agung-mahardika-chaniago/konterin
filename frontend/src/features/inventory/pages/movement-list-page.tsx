import * as React from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { ArrowLeft, ChevronLeft, ChevronRight, History } from 'lucide-react';

import { EmptyState } from '@/components/shared/empty-state';
import { Alert } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select } from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { formatCurrency } from '@/features/dashboard/format';
import { hasPermission } from '@/lib/permissions';
import { useAuthStore } from '@/stores/auth-store';
import { listMovements, listSpareparts } from '../api';
import { formatDateTime, movementTypeLabel, movementTypeVariant } from '../format';
import type { Movement, Sparepart } from '../types';

const MOVEMENT_TYPES = [
  'initial',
  'purchase_in',
  'service_out',
  'service_return',
  'adjustment_in',
  'adjustment_out',
  'transfer_in',
  'transfer_out',
];

const REFERENCE_LABELS: Record<string, string> = {
  service_order: 'Servis',
  purchase: 'Pembelian',
  opname: 'Opname',
  transfer: 'Transfer',
};

export function MovementListPage() {
  const [searchParams] = useSearchParams();
  const user = useAuthStore((state) => state.user);
  const canViewCost = hasPermission(user, 'inventory.cost.view');

  const [items, setItems] = React.useState<Movement[]>([]);
  const [parts, setParts] = React.useState<Sparepart[]>([]);
  const [total, setTotal] = React.useState(0);
  const [page, setPage] = React.useState(1);
  const [totalPages, setTotalPages] = React.useState(1);
  const [sparepartId, setSparepartId] = React.useState(searchParams.get('sparepart_id') ?? '');
  const [movementType, setMovementType] = React.useState('');
  const [from, setFrom] = React.useState('');
  const [to, setTo] = React.useState('');
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);

  React.useEffect(() => {
    listSpareparts({ per_page: 100 })
      .then((result) => setParts(result.data ?? []))
      .catch(() => undefined);
  }, []);

  const loadData = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await listMovements({
        sparepart_id: sparepartId || undefined,
        movement_type: movementType || undefined,
        from: from || undefined,
        to: to || undefined,
        page,
        per_page: 20,
      });
      setItems(result.data ?? []);
      if (result.meta) {
        setTotal(result.meta.total);
        setTotalPages(result.meta.total_pages || 1);
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Gagal memuat pergerakan stok.');
    } finally {
      setLoading(false);
    }
  }, [sparepartId, movementType, from, to, page]);

  React.useEffect(() => {
    loadData();
  }, [loadData]);

  return (
    <div className="space-y-5">
      <div className="flex items-start gap-3">
        <Link to="/spareparts">
          <Button variant="ghost" size="icon" title="Kembali">
            <ArrowLeft className="h-4 w-4" aria-hidden="true" />
          </Button>
        </Link>
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-ink sm:text-[28px]">Pergerakan Stok</h1>
          <p className="mt-1 text-sm text-muted">
            Jejak append-only semua stok masuk, keluar, pembelian, dan penyesuaian.
          </p>
        </div>
      </div>

      {error && <Alert variant="error">{error}</Alert>}

      <Card className="p-4">
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <div className="space-y-1.5">
            <Label htmlFor="movement-part">Sparepart</Label>
            <Select
              id="movement-part"
              value={sparepartId}
              onChange={(e) => {
                setSparepartId(e.target.value);
                setPage(1);
              }}
            >
              <option value="">Semua sparepart</option>
              {parts.map((part) => (
                <option key={part.id} value={part.id}>
                  {part.name} ({part.sku})
                </option>
              ))}
            </Select>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="movement-type">Tipe</Label>
            <Select
              id="movement-type"
              value={movementType}
              onChange={(e) => {
                setMovementType(e.target.value);
                setPage(1);
              }}
            >
              <option value="">Semua tipe</option>
              {MOVEMENT_TYPES.map((type) => (
                <option key={type} value={type}>
                  {movementTypeLabel(type)}
                </option>
              ))}
            </Select>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="movement-from">Dari Tanggal</Label>
            <Input
              id="movement-from"
              type="date"
              value={from}
              onChange={(e) => {
                setFrom(e.target.value);
                setPage(1);
              }}
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="movement-to">Sampai Tanggal</Label>
            <Input
              id="movement-to"
              type="date"
              value={to}
              onChange={(e) => {
                setTo(e.target.value);
                setPage(1);
              }}
            />
          </div>
        </div>
      </Card>

      <Card className="overflow-hidden">
        {loading ? (
          <div className="space-y-3 p-4">
            {[1, 2, 3, 4, 5].map((i) => (
              <div key={i} className="flex items-center justify-between py-2">
                <Skeleton className="h-4 w-52" />
                <Skeleton className="h-4 w-20" />
              </div>
            ))}
          </div>
        ) : items.length === 0 ? (
          <EmptyState
            icon={History}
            title="Belum ada pergerakan stok"
            description="Pergerakan muncul otomatis saat pembelian, pemakaian di servis, atau penyesuaian stok."
            className="py-12"
          />
        ) : (
          <>
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead className="border-b border-border bg-background text-xs uppercase tracking-wider text-muted">
                  <tr>
                    <th scope="col" className="px-4 py-3 font-semibold">Waktu</th>
                    <th scope="col" className="px-4 py-3 font-semibold">Sparepart</th>
                    <th scope="col" className="px-4 py-3 font-semibold">Tipe</th>
                    <th scope="col" className="px-4 py-3 text-right font-semibold">Qty</th>
                    <th scope="col" className="px-4 py-3 text-right font-semibold">Stok</th>
                    {canViewCost && <th scope="col" className="px-4 py-3 text-right font-semibold">Modal</th>}
                    <th scope="col" className="px-4 py-3 font-semibold">Outlet</th>
                    <th scope="col" className="px-4 py-3 font-semibold">Referensi</th>
                    <th scope="col" className="px-4 py-3 font-semibold">Catatan</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {items.map((movement) => (
                    <tr key={movement.id} className="transition-colors hover:bg-background/50">
                      <td className="whitespace-nowrap px-4 py-3 text-xs text-muted">
                        {formatDateTime(movement.created_at)}
                      </td>
                      <td className="px-4 py-3">
                        <Link
                          to={`/spareparts/${movement.sparepart_id}`}
                          className="text-sm font-medium text-ink transition-colors hover:text-accent-600"
                        >
                          {movement.sparepart_name}
                        </Link>
                        <p className="text-xs text-muted">{movement.sku}</p>
                      </td>
                      <td className="px-4 py-3">
                        <Badge variant={movementTypeVariant(movement.movement_type)}>
                          {movementTypeLabel(movement.movement_type)}
                        </Badge>
                      </td>
                      <td
                        className={`whitespace-nowrap px-4 py-3 text-right text-xs font-semibold ${
                          movement.qty > 0 ? 'text-emerald-600' : 'text-red-600'
                        }`}
                      >
                        {movement.qty > 0 ? `+${movement.qty}` : movement.qty}
                      </td>
                      <td className="whitespace-nowrap px-4 py-3 text-right text-xs text-ink">
                        {movement.stock_before} → {movement.stock_after}
                      </td>
                      {canViewCost && (
                        <td className="whitespace-nowrap px-4 py-3 text-right text-xs text-muted">
                          {formatCurrency(movement.unit_cost)}
                        </td>
                      )}
                      <td className="px-4 py-3 text-xs text-muted">{movement.outlet_name}</td>
                      <td className="px-4 py-3 text-xs text-muted">
                        {REFERENCE_LABELS[movement.reference_type] ?? movement.reference_type}
                        {movement.reference_id ? ` · ${movement.reference_id.slice(0, 8)}` : ''}
                      </td>
                      <td className="max-w-[220px] px-4 py-3 text-xs text-muted">
                        <span className="line-clamp-2" title={movement.notes ?? ''}>
                          {movement.notes || '—'}
                        </span>
                        {movement.created_by && <p className="mt-0.5 text-[11px]">oleh {movement.created_by}</p>}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <div className="flex items-center justify-between border-t border-border px-5 py-3 text-xs text-muted">
              <span>
                Menampilkan {items.length} dari total {total} pergerakan
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
    </div>
  );
}
