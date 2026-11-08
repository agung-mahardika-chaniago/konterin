import * as React from 'react';
import { Link } from 'react-router-dom';
import { ChevronLeft, ChevronRight, Eye, ReceiptText, Search, Wallet } from 'lucide-react';

import { EmptyState } from '@/components/shared/empty-state';
import { Alert } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Select } from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { formatCurrency } from '@/features/dashboard/format';
import { hasPermission } from '@/lib/permissions';
import { useAuthStore } from '@/stores/auth-store';
import { listInvoices } from '../api';
import { InvoiceStatusBadge } from '../components/invoice-status-badge';
import { INVOICE_STATUS, formatDateTime } from '../format';
import type { InvoiceListItem, InvoiceStatus } from '../types';

const STATUS_OPTIONS = Object.keys(INVOICE_STATUS) as InvoiceStatus[];

export function InvoiceListPage() {
  const user = useAuthStore((state) => state.user);
  const canOpenCashier = hasPermission(user, 'payment.view');

  const [items, setItems] = React.useState<InvoiceListItem[]>([]);
  const [total, setTotal] = React.useState(0);
  const [page, setPage] = React.useState(1);
  const [totalPages, setTotalPages] = React.useState(1);
  const [search, setSearch] = React.useState('');
  const [debouncedSearch, setDebouncedSearch] = React.useState('');
  const [status, setStatus] = React.useState('');
  const [from, setFrom] = React.useState('');
  const [to, setTo] = React.useState('');
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);

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
      const result = await listInvoices({
        q: debouncedSearch.trim() || undefined,
        status: status || undefined,
        from: from || undefined,
        to: to || undefined,
        page,
        per_page: 15,
      });
      setItems(result.data ?? []);
      if (result.meta) {
        setTotal(result.meta.total);
        setTotalPages(result.meta.total_pages || 1);
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Gagal memuat daftar invoice.');
    } finally {
      setLoading(false);
    }
  }, [debouncedSearch, status, from, to, page]);

  React.useEffect(() => {
    loadData();
  }, [loadData]);

  const pageTotal = items.reduce((sum, item) => sum + item.total_amount, 0);
  const pageOutstanding = items.reduce((sum, item) => sum + item.remaining_amount, 0);
  const hasFilter = Boolean(search || status || from || to);

  const resetFilters = () => {
    setSearch('');
    setStatus('');
    setFrom('');
    setTo('');
    setPage(1);
  };

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-ink sm:text-[28px]">Invoice</h1>
          <p className="mt-1 text-sm text-muted">
            Pantau status tagihan, diskon, pembayaran, dan cetak invoice pelanggan.
          </p>
        </div>
        {canOpenCashier && (
          <Link to="/cashier">
            <Button className="gap-2">
              <ReceiptText className="h-4 w-4" aria-hidden="true" />
              Buka Kasir
            </Button>
          </Link>
        )}
      </div>

      {error && <Alert variant="error">{error}</Alert>}

      <div className="grid gap-4 sm:grid-cols-3">
        <Card className="p-4">
          <p className="text-xs font-medium uppercase tracking-wide text-muted">Total Invoice</p>
          <p className="mt-1 text-2xl font-bold text-ink">{total}</p>
          <p className="text-xs text-muted">sesuai filter aktif</p>
        </Card>
        <Card className="p-4">
          <p className="text-xs font-medium uppercase tracking-wide text-muted">Nilai Tagihan</p>
          <p className="mt-1 text-2xl font-bold text-ink">{formatCurrency(pageTotal)}</p>
          <p className="text-xs text-muted">halaman ini</p>
        </Card>
        <Card className="p-4">
          <p className="text-xs font-medium uppercase tracking-wide text-muted">Sisa Tagihan</p>
          <p className="mt-1 text-2xl font-bold text-amber-700">{formatCurrency(pageOutstanding)}</p>
          <p className="text-xs text-muted">halaman ini</p>
        </Card>
      </div>

      <div className="flex flex-col gap-3 lg:flex-row lg:items-center">
        <div className="relative flex-1 lg:max-w-md">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted" aria-hidden="true" />
          <Input
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder="Cari nomor invoice, tiket, pelanggan, atau telepon..."
            className="pl-9"
          />
        </div>
        <Select
          value={status}
          onChange={(event) => {
            setStatus(event.target.value);
            setPage(1);
          }}
          className="lg:max-w-[180px]"
        >
          <option value="">Semua status</option>
          {STATUS_OPTIONS.map((value) => (
            <option key={value} value={value}>
              {INVOICE_STATUS[value].label}
            </option>
          ))}
        </Select>
        <div className="flex items-center gap-2">
          <Input
            type="date"
            value={from}
            onChange={(event) => {
              setFrom(event.target.value);
              setPage(1);
            }}
            className="w-[145px]"
            title="Tanggal awal"
          />
          <span className="text-xs text-muted">s.d.</span>
          <Input
            type="date"
            value={to}
            onChange={(event) => {
              setTo(event.target.value);
              setPage(1);
            }}
            className="w-[145px]"
            title="Tanggal akhir"
          />
        </div>
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
            icon={Wallet}
            title="Belum ada invoice"
            description={
              hasFilter
                ? 'Tidak ada invoice yang sesuai dengan filter Anda.'
                : 'Invoice dibuat otomatis saat servis diterima. Finalisasi servis untuk melihat tagihan.'
            }
            action={
              <Link to="/services">
                <Button variant="outline" size="sm">
                  Lihat Daftar Servis
                </Button>
              </Link>
            }
            className="py-12"
          />
        ) : (
          <>
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead className="border-b border-border bg-background text-xs uppercase tracking-wider text-muted">
                  <tr>
                    <th scope="col" className="px-5 py-3 font-semibold">Invoice</th>
                    <th scope="col" className="px-5 py-3 font-semibold">Pelanggan</th>
                    <th scope="col" className="px-5 py-3 font-semibold">Tiket Servis</th>
                    <th scope="col" className="px-5 py-3 text-right font-semibold">Total</th>
                    <th scope="col" className="px-5 py-3 text-right font-semibold">Dibayar</th>
                    <th scope="col" className="px-5 py-3 text-right font-semibold">Sisa</th>
                    <th scope="col" className="px-5 py-3 font-semibold">Tanggal</th>
                    <th scope="col" className="px-5 py-3 text-right font-semibold">Aksi</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {items.map((item) => (
                    <tr key={item.id} className="transition-colors hover:bg-background/50">
                      <td className="px-5 py-3.5">
                        <Link
                          to={`/invoices/${item.id}`}
                          className="font-mono text-xs font-bold text-ink transition-colors hover:text-accent-600"
                        >
                          {item.invoice_no}
                        </Link>
                        <div className="mt-1">
                          <InvoiceStatusBadge status={item.status} />
                        </div>
                      </td>
                      <td className="px-5 py-3.5">
                        <p className="font-medium text-ink">{item.customer_name}</p>
                        <p className="text-xs text-muted font-mono">{item.customer_phone}</p>
                      </td>
                      <td className="px-5 py-3.5">
                        <Link to={`/services/${item.service_order_id}`} className="text-xs text-accent-600 hover:underline">
                          {item.ticket_no}
                        </Link>
                        <p className="text-xs text-muted">{item.outlet_name}</p>
                      </td>
                      <td className="px-5 py-3.5 text-right font-mono text-xs font-semibold text-ink">
                        {formatCurrency(item.total_amount)}
                      </td>
                      <td className="px-5 py-3.5 text-right font-mono text-xs text-emerald-700">
                        {formatCurrency(item.paid_amount)}
                      </td>
                      <td className="px-5 py-3.5 text-right font-mono text-xs font-semibold text-amber-700">
                        {formatCurrency(item.remaining_amount)}
                      </td>
                      <td className="px-5 py-3.5 text-xs text-muted">{formatDateTime(item.created_at)}</td>
                      <td className="whitespace-nowrap px-5 py-3.5 text-right">
                        <Link to={`/invoices/${item.id}`}>
                          <Button variant="ghost" size="sm" className="h-8 gap-1 px-2" title="Buka detail invoice">
                            <Eye className="h-3.5 w-3.5" aria-hidden="true" />
                            Detail
                          </Button>
                        </Link>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            <div className="flex items-center justify-between border-t border-border px-5 py-3 text-xs text-muted">
              <span>
                Menampilkan {items.length} dari total {total} invoice
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
    </div>
  );
}
