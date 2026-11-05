import * as React from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import {
  ChevronLeft,
  ChevronRight,
  Eye,
  Plus,
  Printer,
  RotateCcw,
  Search,
  Wrench,
} from 'lucide-react';

import { EmptyState } from '@/components/shared/empty-state';
import { Alert } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Select } from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { formatCurrency } from '@/features/dashboard/format';
import { PrintLabelModal } from '@/features/print/print-label-modal';
import { useOutletStore } from '@/stores/outlet-store';
import { getTechnicians, listServiceOrders } from '../api';
import { PriorityBadge } from '../components/priority-badge';
import { StatusBadge } from '../components/status-badge';
import type { ServiceOrderListItem, TechnicianOption } from '../types';

export function ServiceListPage() {
  const [searchParams] = useSearchParams();
  const outletId = useOutletStore((state) => state.outletId);

  const [orders, setOrders] = React.useState<ServiceOrderListItem[]>([]);
  const [total, setTotal] = React.useState(0);
  const [page, setPage] = React.useState(1);
  const [totalPages, setTotalPages] = React.useState(1);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);

  // Filters
  const [search, setSearch] = React.useState(searchParams.get('q') || '');
  const [debouncedSearch, setDebouncedSearch] = React.useState(searchParams.get('q') || '');
  const [status, setStatus] = React.useState(searchParams.get('status') || '');
  const [technicianId, setTechnicianId] = React.useState(searchParams.get('technician_id') || '');
  const [fromDate, setFromDate] = React.useState(searchParams.get('from') || '');
  const [toDate, setToDate] = React.useState(searchParams.get('to') || '');

  const [technicians, setTechnicians] = React.useState<TechnicianOption[]>([]);

  // Print Label Modal
  const [printOrder, setPrintOrder] = React.useState<ServiceOrderListItem | null>(null);
  const [printModalOpen, setPrintModalOpen] = React.useState(false);

  React.useEffect(() => {
    getTechnicians(outletId ?? undefined)
      .then(setTechnicians)
      .catch((err) => console.error('Error fetching technicians', err));
  }, [outletId]);

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
      const res = await listServiceOrders({
        q: debouncedSearch.trim() || undefined,
        status: status || undefined,
        technician_id: technicianId || undefined,
        outlet_id: outletId ?? undefined,
        from: fromDate || undefined,
        to: toDate || undefined,
        page,
        per_page: 15,
      });
      setOrders(res.data ?? []);
      if (res.meta) {
        setTotal(res.meta.total);
        setTotalPages(res.meta.total_pages || 1);
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Gagal memuat daftar servis.');
    } finally {
      setLoading(false);
    }
  }, [debouncedSearch, status, technicianId, outletId, fromDate, toDate, page]);

  React.useEffect(() => {
    loadData();
  }, [loadData]);

  const handleResetFilters = () => {
    setSearch('');
    setStatus('');
    setTechnicianId('');
    setFromDate('');
    setToDate('');
    setPage(1);
  };

  const handleOpenPrint = (order: ServiceOrderListItem) => {
    setPrintOrder(order);
    setPrintModalOpen(true);
  };

  const hasActiveFilters = Boolean(search || status || technicianId || fromDate || toDate);

  return (
    <div className="space-y-5">
      {/* Page Header */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-ink sm:text-[28px]">Servis HP</h1>
          <p className="mt-1 text-sm text-muted">
            Kelola penerimaan servis, perbaikan teknisi, dan cetak label tiket.
          </p>
        </div>
        <Link to="/services/new" className="self-start sm:self-auto">
          <Button className="gap-2 shadow-sm">
            <Plus className="h-4 w-4" aria-hidden="true" />
            Servis Baru
          </Button>
        </Link>
      </div>

      {error && <Alert variant="error">{error}</Alert>}

      {/* Filter Toolbar */}
      <Card className="p-4 space-y-3">
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
          {/* Search */}
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted" aria-hidden="true" />
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Cari tiket, nama, nomor HP, tipe..."
              className="pl-9"
            />
          </div>

          {/* Status Filter */}
          <div>
            <Select
              value={status}
              onChange={(e) => {
                setStatus(e.target.value);
                setPage(1);
              }}
            >
              <option value="">Semua Status</option>
              <option value="received">Diterima</option>
              <option value="waiting_diagnosis">Menunggu Diagnosis</option>
              <option value="waiting_approval">Menunggu Persetujuan</option>
              <option value="waiting_part">Menunggu Sparepart</option>
              <option value="in_progress">Sedang Diperbaiki</option>
              <option value="testing">Testing</option>
              <option value="completed">Selesai</option>
              <option value="ready_for_pickup">Siap Diambil</option>
              <option value="picked_up">Sudah Diambil</option>
              <option value="cancelled">Dibatalkan</option>
            </Select>
          </div>

          {/* Technician Filter */}
          <div>
            <Select
              value={technicianId}
              onChange={(e) => {
                setTechnicianId(e.target.value);
                setPage(1);
              }}
            >
              <option value="">Semua Teknisi</option>
              {technicians.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.name}
                </option>
              ))}
            </Select>
          </div>

          {/* Date from & to */}
          <div className="flex items-center gap-2">
            <Input
              type="date"
              value={fromDate}
              onChange={(e) => {
                setFromDate(e.target.value);
                setPage(1);
              }}
              title="Dari tanggal"
              className="text-xs px-2"
            />
            <span className="text-muted text-xs">-</span>
            <Input
              type="date"
              value={toDate}
              onChange={(e) => {
                setToDate(e.target.value);
                setPage(1);
              }}
              title="Sampai tanggal"
              className="text-xs px-2"
            />
          </div>
        </div>

        {hasActiveFilters && (
          <div className="flex items-center justify-between pt-1 border-t border-border text-xs">
            <span className="text-muted">Filter aktif diterapkan</span>
            <button
              type="button"
              onClick={handleResetFilters}
              className="inline-flex items-center gap-1 font-medium text-accent-600 hover:text-accent-700"
            >
              <RotateCcw className="h-3 w-3" aria-hidden="true" />
              Reset filter
            </button>
          </div>
        )}
      </Card>

      {/* Main Table / List */}
      <Card className="overflow-hidden">
        {loading ? (
          <div className="divide-y divide-border p-4 space-y-3">
            {[1, 2, 3, 4, 5].map((i) => (
              <div key={i} className="flex items-center justify-between py-2">
                <div className="space-y-1">
                  <Skeleton className="h-4 w-40" />
                  <Skeleton className="h-3 w-60" />
                </div>
                <Skeleton className="h-8 w-24" />
              </div>
            ))}
          </div>
        ) : orders.length === 0 ? (
          <EmptyState
            icon={Wrench}
            title="Tidak ada pesanan servis"
            description={
              hasActiveFilters
                ? 'Tidak ada pesanan servis yang cocok dengan filter yang dipilih.'
                : 'Mulai dengan membuat tanda terima penerimaan servis baru.'
            }
            action={
              <Link to="/services/new">
                <Button size="sm" className="gap-1.5">
                  <Plus className="h-4 w-4" aria-hidden="true" />
                  Terima Servis Baru
                </Button>
              </Link>
            }
            className="py-12"
          />
        ) : (
          <>
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead className="bg-background text-xs uppercase tracking-wider text-muted border-b border-border">
                  <tr>
                    <th scope="col" className="px-5 py-3 font-semibold">Tiket & Tanggal</th>
                    <th scope="col" className="px-5 py-3 font-semibold">Pelanggan</th>
                    <th scope="col" className="px-5 py-3 font-semibold">Perangkat & Keluhan</th>
                    <th scope="col" className="px-5 py-3 font-semibold">Status</th>
                    <th scope="col" className="px-5 py-3 font-semibold">Teknisi</th>
                    <th scope="col" className="px-5 py-3 font-semibold text-right">Estimasi</th>
                    <th scope="col" className="px-5 py-3 font-semibold text-right">Aksi</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {orders.map((o) => (
                    <tr key={o.id} className="hover:bg-background/50 transition-colors">
                      <td className="px-5 py-3.5 whitespace-nowrap">
                        <div className="flex items-center gap-2">
                          <Link
                            to={`/services/${o.id}`}
                            className="font-bold font-mono text-ink hover:text-accent-600 transition-colors"
                          >
                            {o.ticket_no}
                          </Link>
                          <PriorityBadge priority={o.priority} />
                        </div>
                        <p className="text-xs text-muted mt-0.5">
                          {new Date(o.received_at).toLocaleDateString('id-ID', {
                            day: 'numeric',
                            month: 'short',
                            year: 'numeric',
                            hour: '2-digit',
                            minute: '2-digit',
                          })}
                        </p>
                      </td>
                      <td className="px-5 py-3.5">
                        <p className="font-semibold text-ink">{o.customer.name}</p>
                        <p className="text-xs text-muted font-mono">{o.customer.phone}</p>
                      </td>
                      <td className="px-5 py-3.5 max-w-xs">
                        <p className="font-medium text-ink">
                          {o.device.brand} {o.device.model}
                          {o.device.color ? ` (${o.device.color})` : ''}
                        </p>
                        <p className="text-xs text-muted line-clamp-1 italic mt-0.5">{o.complaint}</p>
                      </td>
                      <td className="px-5 py-3.5 whitespace-nowrap">
                        <StatusBadge status={o.status} />
                      </td>
                      <td className="px-5 py-3.5 whitespace-nowrap text-xs text-ink">
                        {o.assigned_technician ? (
                          <span className="font-medium">{o.assigned_technician.name}</span>
                        ) : (
                          <span className="text-muted italic">Belum ditugaskan</span>
                        )}
                      </td>
                      <td className="px-5 py-3.5 text-right whitespace-nowrap">
                        <p className="font-semibold text-ink text-xs">{formatCurrency(o.estimated_cost)}</p>
                        {o.estimated_done_at && (
                          <p className="text-[11px] text-muted">
                            Selesai:{' '}
                            {new Date(o.estimated_done_at).toLocaleDateString('id-ID', {
                              day: 'numeric',
                              month: 'short',
                            })}
                          </p>
                        )}
                      </td>
                      <td className="px-5 py-3.5 text-right whitespace-nowrap">
                        <div className="flex items-center justify-end gap-1">
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleOpenPrint(o)}
                            className="h-8 px-2 text-xs"
                            title="Cetak label tiket"
                          >
                            <Printer className="h-3.5 w-3.5 text-muted hover:text-ink" aria-hidden="true" />
                            <span className="sr-only">Cetak label</span>
                          </Button>
                          <Link to={`/services/${o.id}`}>
                            <Button variant="ghost" size="sm" className="h-8 px-2 text-xs" title="Lihat detail">
                              <Eye className="h-3.5 w-3.5 text-muted hover:text-ink" aria-hidden="true" />
                              <span className="sr-only">Detail</span>
                            </Button>
                          </Link>
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
                Menampilkan {orders.length} dari total {total} servis
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

      {/* Print Label Modal */}
      <PrintLabelModal
        open={printModalOpen}
        onOpenChange={setPrintModalOpen}
        order={printOrder}
      />
    </div>
  );
}
