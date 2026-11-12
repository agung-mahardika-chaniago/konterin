import * as React from 'react';
import {
  AlertTriangle,
  Boxes,
  ClipboardList,
  Coins,
  ReceiptText,
  Smartphone,
  TrendingUp,
  Users,
  Wrench,
} from 'lucide-react';

import { EmptyState } from '@/components/shared/empty-state';
import { Alert } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Card } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { formatCurrency, formatNumber } from '@/features/dashboard/format';
import { itemTypeLabel } from '@/features/invoices/format';
import { STATUS_CONFIG } from '@/features/services/components/status-badge';
import {
  getBrandsReport,
  getExpensesReport,
  getInventoryValuation,
  getProfitReport,
  getRevenueReport,
  getServiceVsPartsReport,
  getServicesReport,
  getStockReport,
  getTechniciansReport,
} from '../api';
import { formatHours, formatReportDate } from '../format';
import type { ReportFilter, ReportPeriod } from '../types';
import { RevenueChart } from './revenue-chart';

function useReportLoader<T>(
  load: () => Promise<T>,
  deps: React.DependencyList,
): { data: T | null; loading: boolean; error: string | null } {
  const [data, setData] = React.useState<T | null>(null);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);

  React.useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);
    load()
      .then((result) => {
        if (!cancelled) setData(result);
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof Error ? err.message : 'Gagal memuat laporan.');
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);

  return { data, loading, error };
}

function ReportSkeleton() {
  return (
    <Card className="space-y-3 p-4">
      {[1, 2, 3, 4, 5].map((i) => (
        <div key={i} className="flex items-center justify-between py-2">
          <Skeleton className="h-4 w-48" />
          <Skeleton className="h-4 w-24" />
        </div>
      ))}
    </Card>
  );
}

function RangeNote({ from, to }: { from: string; to: string }) {
  return (
    <p className="text-xs text-muted">
      Rentang {formatReportDate(from)} – {formatReportDate(to)} (zona waktu outlet)
    </p>
  );
}

function StatCard({ label, value, hint }: { label: string; value: string; hint?: string }) {
  return (
    <Card className="p-4">
      <p className="text-xs font-medium uppercase tracking-wide text-muted">{label}</p>
      <p className="mt-1 text-2xl font-bold text-ink">{value}</p>
      {hint && <p className="text-xs text-muted">{hint}</p>}
    </Card>
  );
}

function TableShell({ children }: { children: React.ReactNode }) {
  return (
    <Card className="overflow-hidden">
      <div className="overflow-x-auto">
        <table className="w-full text-left text-sm">{children}</table>
      </div>
    </Card>
  );
}

function TableHead({ cells }: { cells: (string | React.ReactNode)[] }) {
  return (
    <thead className="border-b border-border bg-background text-xs uppercase tracking-wider text-muted">
      <tr>
        {cells.map((cell, index) => (
          <th key={index} scope="col" className="px-5 py-3 font-semibold">
            {cell}
          </th>
        ))}
      </tr>
    </thead>
  );
}

export function RevenueSection({ filter, period }: { filter: ReportFilter; period: ReportPeriod }) {
  const { data, loading, error } = useReportLoader(() => getRevenueReport({ ...filter, period }), [filter, period]);

  if (loading) return <ReportSkeleton />;
  if (error) return <Alert variant="error">{error}</Alert>;
  if (!data) return null;

  return (
    <div className="space-y-4">
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard label="Total Pendapatan" value={formatCurrency(data.total.total_revenue)} hint={`${formatNumber(data.total.invoice_count)} invoice`} />
        <StatCard label="Sudah Dibayar" value={formatCurrency(data.total.paid_amount)} />
        <StatCard label="Belum Dibayar" value={formatCurrency(data.total.unpaid_amount)} />
        <StatCard label="Jumlah Servis" value={formatNumber(data.total.service_count)} />
      </div>

      <Card className="p-4">
        <div className="mb-2 flex items-baseline justify-between">
          <h2 className="text-sm font-semibold text-ink">Tren Pendapatan</h2>
          <RangeNote from={data.from} to={data.to} />
        </div>
        <RevenueChart points={data.items} />
      </Card>

      {data.items.length === 0 ? (
        <EmptyState
          icon={Coins}
          title="Belum ada data pendapatan"
          description="Terbitkan invoice pertama agar pendapatan mulai tercatat di laporan."
          className="py-12"
        />
      ) : (
        <TableShell>
          <TableHead
            cells={[
              'Periode',
              'Invoice',
              'Servis',
              'Jasa',
              'Part',
              'Diskon',
              'Total',
              'Terbayar',
              'Belum Dibayar',
            ]}
          />
          <tbody className="divide-y divide-border">
            {data.items.map((point) => (
              <tr key={point.label} className="transition-colors hover:bg-background/50">
                <td className="px-5 py-3.5 font-medium text-ink">{point.label}</td>
                <td className="px-5 py-3.5 text-xs text-muted">{formatNumber(point.invoice_count)}</td>
                <td className="px-5 py-3.5 text-xs text-muted">{formatNumber(point.service_count)}</td>
                <td className="px-5 py-3.5 text-right font-mono text-xs text-ink">{formatCurrency(point.service_amount)}</td>
                <td className="px-5 py-3.5 text-right font-mono text-xs text-ink">{formatCurrency(point.part_amount)}</td>
                <td className="px-5 py-3.5 text-right font-mono text-xs text-muted">-{formatCurrency(point.discount_amount)}</td>
                <td className="px-5 py-3.5 text-right font-mono text-xs font-semibold text-ink">{formatCurrency(point.total_revenue)}</td>
                <td className="px-5 py-3.5 text-right font-mono text-xs text-muted">{formatCurrency(point.paid_amount)}</td>
                <td className="px-5 py-3.5 text-right font-mono text-xs text-muted">{formatCurrency(point.unpaid_amount)}</td>
              </tr>
            ))}
            <tr className="border-t border-border bg-background font-semibold">
              <td className="px-5 py-3 text-xs uppercase tracking-wide text-muted">Total</td>
              <td className="px-5 py-3 text-xs text-ink">{formatNumber(data.total.invoice_count)}</td>
              <td className="px-5 py-3 text-xs text-ink">{formatNumber(data.total.service_count)}</td>
              <td className="px-5 py-3 text-right font-mono text-xs text-ink">{formatCurrency(data.total.service_amount)}</td>
              <td className="px-5 py-3 text-right font-mono text-xs text-ink">{formatCurrency(data.total.part_amount)}</td>
              <td className="px-5 py-3 text-right font-mono text-xs text-muted">-{formatCurrency(data.total.discount_amount)}</td>
              <td className="px-5 py-3 text-right font-mono text-xs text-ink">{formatCurrency(data.total.total_revenue)}</td>
              <td className="px-5 py-3 text-right font-mono text-xs text-ink">{formatCurrency(data.total.paid_amount)}</td>
              <td className="px-5 py-3 text-right font-mono text-xs text-ink">{formatCurrency(data.total.unpaid_amount)}</td>
            </tr>
          </tbody>
        </TableShell>
      )}
    </div>
  );
}

export function ServicesSection({ filter }: { filter: ReportFilter }) {
  const { data, loading, error } = useReportLoader(() => getServicesReport(filter), [filter]);

  if (loading) return <ReportSkeleton />;
  if (error) return <Alert variant="error">{error}</Alert>;
  if (!data) return null;

  return (
    <div className="space-y-4">
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
        <StatCard label="Total Servis" value={formatNumber(data.total)} />
        <StatCard label="Selesai" value={formatNumber(data.completed)} />
        <StatCard label="Sedang Berlangsung" value={formatNumber(data.in_progress)} />
        <StatCard label="Dibatalkan" value={formatNumber(data.cancelled)} />
        <StatCard label="Rata-rata Durasi" value={formatHours(data.avg_hours)} hint="dari diterima sampai selesai" />
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <div>
          <h2 className="mb-2 text-sm font-semibold text-ink">Rincian Status</h2>
          <TableShell>
            <TableHead cells={['Status', 'Jumlah']} />
            <tbody className="divide-y divide-border">
              {data.status_breakdown.length === 0 ? (
                <tr>
                  <td colSpan={2}>
                    <EmptyState
                      icon={ClipboardList}
                      title="Belum ada servis pada rentang ini"
                      description="Coba ubah rentang tanggal atau catat servis baru."
                      className="py-10"
                    />
                  </td>
                </tr>
              ) : (
                data.status_breakdown.map((row) => (
                  <tr key={row.status} className="transition-colors hover:bg-background/50">
                    <td className="px-5 py-3.5">
                      <Badge variant="neutral">{STATUS_CONFIG[row.status]?.label ?? row.status}</Badge>
                    </td>
                    <td className="px-5 py-3.5 text-right font-mono text-xs text-ink">{formatNumber(row.count)}</td>
                  </tr>
                ))
              )}
            </tbody>
          </TableShell>
        </div>
        <div>
          <h2 className="mb-2 text-sm font-semibold text-ink">Servis Selesai per Teknisi</h2>
          <TableShell>
            <TableHead cells={['Teknisi', 'Servis Selesai']} />
            <tbody className="divide-y divide-border">
              {data.technician_rank.length === 0 ? (
                <tr>
                  <td colSpan={2}>
                    <EmptyState
                      icon={Users}
                      title="Belum ada servis selesai"
                      description="Servis yang ditandai selesai oleh teknisi akan muncul di sini."
                      className="py-10"
                    />
                  </td>
                </tr>
              ) : (
                data.technician_rank.map((row) => (
                  <tr key={row.name} className="transition-colors hover:bg-background/50">
                    <td className="px-5 py-3.5 font-medium text-ink">{row.name}</td>
                    <td className="px-5 py-3.5 text-right font-mono text-xs text-ink">{formatNumber(row.count)}</td>
                  </tr>
                ))
              )}
            </tbody>
          </TableShell>
        </div>
      </div>
      <RangeNote from={data.from} to={data.to} />
    </div>
  );
}

export function BrandsSection({ filter }: { filter: ReportFilter }) {
  const { data, loading, error } = useReportLoader(() => getBrandsReport(filter), [filter]);

  if (loading) return <ReportSkeleton />;
  if (error) return <Alert variant="error">{error}</Alert>;
  if (!data) return null;

  const totalService = data.items.reduce((sum, item) => sum + item.count, 0);
  const totalRevenue = data.items.reduce((sum, item) => sum + item.revenue, 0);

  return (
    <div className="space-y-4">
      {data.items.length === 0 ? (
        <EmptyState
          icon={Smartphone}
          title="Belum ada data merek"
          description="Belum ada servis dengan merek perangkat pada rentang ini."
          className="py-12"
        />
      ) : (
        <TableShell>
          <TableHead cells={['Merek', 'Jumlah Servis', 'Pendapatan']} />
          <tbody className="divide-y divide-border">
            {data.items.map((item) => (
              <tr key={item.brand} className="transition-colors hover:bg-background/50">
                <td className="px-5 py-3.5 font-medium text-ink">{item.brand}</td>
                <td className="px-5 py-3.5 text-right font-mono text-xs text-ink">{formatNumber(item.count)}</td>
                <td className="px-5 py-3.5 text-right font-mono text-xs font-semibold text-ink">{formatCurrency(item.revenue)}</td>
              </tr>
            ))}
            <tr className="border-t border-border bg-background font-semibold">
              <td className="px-5 py-3 text-xs uppercase tracking-wide text-muted">Total</td>
              <td className="px-5 py-3 text-right font-mono text-xs text-ink">{formatNumber(totalService)}</td>
              <td className="px-5 py-3 text-right font-mono text-xs text-ink">{formatCurrency(totalRevenue)}</td>
            </tr>
          </tbody>
        </TableShell>
      )}
      <RangeNote from={data.from} to={data.to} />
    </div>
  );
}

export function ServiceVsPartsSection({ filter }: { filter: ReportFilter }) {
  const { data, loading, error } = useReportLoader(() => getServiceVsPartsReport(filter), [filter]);

  if (loading) return <ReportSkeleton />;
  if (error) return <Alert variant="error">{error}</Alert>;
  if (!data) return null;

  const totalQty = data.items.reduce((sum, item) => sum + item.qty, 0);
  const totalAmount = data.items.reduce((sum, item) => sum + item.amount, 0);

  return (
    <div className="space-y-4">
      {data.items.length === 0 ? (
        <EmptyState
          icon={Wrench}
          title="Belum ada item terinvoice"
          description="Terbitkan invoice dengan jasa atau sparepart untuk melihat perbandingan ini."
          className="py-12"
        />
      ) : (
        <TableShell>
          <TableHead cells={['Komponen', 'Qty', 'Nominal']} />
          <tbody className="divide-y divide-border">
            {data.items.map((item) => (
              <tr key={item.item_type} className="transition-colors hover:bg-background/50">
                <td className="px-5 py-3.5 font-medium text-ink">{itemTypeLabel(item.item_type)}</td>
                <td className="px-5 py-3.5 text-right font-mono text-xs text-ink">{formatNumber(item.qty)}</td>
                <td className="px-5 py-3.5 text-right font-mono text-xs font-semibold text-ink">{formatCurrency(item.amount)}</td>
              </tr>
            ))}
            <tr className="border-t border-border bg-background font-semibold">
              <td className="px-5 py-3 text-xs uppercase tracking-wide text-muted">Total</td>
              <td className="px-5 py-3 text-right font-mono text-xs text-ink">{formatNumber(totalQty)}</td>
              <td className="px-5 py-3 text-right font-mono text-xs text-ink">{formatCurrency(totalAmount)}</td>
            </tr>
          </tbody>
        </TableShell>
      )}
      <RangeNote from={data.from} to={data.to} />
    </div>
  );
}

export function ProfitSection({ filter }: { filter: ReportFilter }) {
  const { data, loading, error } = useReportLoader(() => getProfitReport(filter), [filter]);

  if (loading) return <ReportSkeleton />;
  if (error) return <Alert variant="error">{error}</Alert>;
  if (!data) return null;

  return (
    <div className="space-y-4">
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard label="Pendapatan" value={formatCurrency(data.revenue)} hint={`${formatNumber(data.invoice_count)} invoice`} />
        <StatCard label="Harga Pokok (HPP)" value={formatCurrency(data.cost)} />
        <StatCard label="Laba Kotor" value={formatCurrency(data.gross_profit)} />
        <StatCard label="Total Pengeluaran" value={formatCurrency(data.expense_total)} />
        <StatCard label="Laba Bersih" value={formatCurrency(data.net_profit)} />
        <StatCard label="Rata-rata per Invoice" value={formatCurrency(data.avg_invoice)} />
        <StatCard
          label="Margin Kotor"
          value={data.revenue > 0 ? `${((data.gross_profit / data.revenue) * 100).toLocaleString('id-ID', { maximumFractionDigits: 1 })}%` : '—'}
        />
        <StatCard
          label="Margin Bersih"
          value={data.revenue > 0 ? `${((data.net_profit / data.revenue) * 100).toLocaleString('id-ID', { maximumFractionDigits: 1 })}%` : '—'}
        />
      </div>
      <RangeNote from={data.from} to={data.to} />
    </div>
  );
}

export function ExpensesSection({ filter }: { filter: ReportFilter }) {
  const { data, loading, error } = useReportLoader(() => getExpensesReport(filter), [filter]);

  if (loading) return <ReportSkeleton />;
  if (error) return <Alert variant="error">{error}</Alert>;
  if (!data) return null;

  return (
    <div className="space-y-4">
      {data.items.length === 0 ? (
        <EmptyState
          icon={ReceiptText}
          title="Belum ada pengeluaran"
          description="Catat pengeluaran operasional agar laporan laba lebih akurat."
          className="py-12"
        />
      ) : (
        <TableShell>
          <TableHead cells={['Kategori', 'Jumlah Transaksi', 'Nominal']} />
          <tbody className="divide-y divide-border">
            {data.items.map((item) => (
              <tr key={item.category} className="transition-colors hover:bg-background/50">
                <td className="px-5 py-3.5">
                  <Badge variant="neutral">{item.category}</Badge>
                </td>
                <td className="px-5 py-3.5 text-right font-mono text-xs text-ink">{formatNumber(item.count)}</td>
                <td className="px-5 py-3.5 text-right font-mono text-xs font-semibold text-ink">{formatCurrency(item.amount)}</td>
              </tr>
            ))}
            <tr className="border-t border-border bg-background font-semibold">
              <td className="px-5 py-3 text-xs uppercase tracking-wide text-muted">Total</td>
              <td className="px-5 py-3" aria-hidden="true" />
              <td className="px-5 py-3 text-right font-mono text-xs text-ink">{formatCurrency(data.total)}</td>
            </tr>
          </tbody>
        </TableShell>
      )}
      <RangeNote from={data.from} to={data.to} />
    </div>
  );
}

export function TechniciansSection({ filter }: { filter: ReportFilter }) {
  const { data, loading, error } = useReportLoader(() => getTechniciansReport(filter), [filter]);

  if (loading) return <ReportSkeleton />;
  if (error) return <Alert variant="error">{error}</Alert>;
  if (!data) return null;

  return (
    <div className="space-y-4">
      {data.items.length === 0 ? (
        <EmptyState
          icon={Users}
          title="Belum ada data teknisi"
          description="Tambahkan teknisi dan tugaskan servis untuk melihat produktivitasnya."
          className="py-12"
        />
      ) : (
        <TableShell>
          <TableHead
            cells={['Teknisi', 'Ditugaskan', 'Selesai', 'Aktif', 'Rata-rata Durasi', 'Kontribusi Pendapatan']}
          />
          <tbody className="divide-y divide-border">
            {data.items.map((item) => (
              <tr key={item.name} className="transition-colors hover:bg-background/50">
                <td className="px-5 py-3.5 font-medium text-ink">{item.name}</td>
                <td className="px-5 py-3.5 text-right font-mono text-xs text-ink">{formatNumber(item.assigned_count)}</td>
                <td className="px-5 py-3.5 text-right font-mono text-xs text-ink">{formatNumber(item.completed_count)}</td>
                <td className="px-5 py-3.5 text-right font-mono text-xs text-ink">{formatNumber(item.active_count)}</td>
                <td className="px-5 py-3.5 text-xs text-ink">{formatHours(item.avg_repair_hours)}</td>
                <td className="px-5 py-3.5 text-right font-mono text-xs font-semibold text-ink">{formatCurrency(item.revenue_contribution)}</td>
              </tr>
            ))}
          </tbody>
        </TableShell>
      )}
      <RangeNote from={data.from} to={data.to} />
    </div>
  );
}

export function StockSection({ filter }: { filter: ReportFilter }) {
  const { data, loading, error } = useReportLoader(() => getStockReport(filter), [filter]);

  if (loading) return <ReportSkeleton />;
  if (error) return <Alert variant="error">{error}</Alert>;
  if (!data) return null;

  return (
    <div className="space-y-4">
      <div className="grid gap-4 sm:grid-cols-3">
        <StatCard label="Nilai Stok" value={formatCurrency(data.total_value)} hint="harga pokok × stok" />
        <StatCard label="Jumlah Item" value={formatNumber(data.total_items)} />
        <StatCard label="Baris Stok Rendah" value={formatNumber(data.low_stock_rows)} hint="stok ≤ minimum" />
      </div>

      {data.items.length === 0 ? (
        <EmptyState
          icon={Boxes}
          title="Belum ada sparepart"
          description="Daftarkan sparepart agar stok dan nilainya terlacak."
          className="py-12"
        />
      ) : (
        <TableShell>
          <TableHead cells={['SKU', 'Sparepart', 'Kategori', 'Stok', 'Minimum', 'HPP', 'Nilai', 'Status']} />
          <tbody className="divide-y divide-border">
            {data.items.map((item) => (
              <tr key={item.sku} className="transition-colors hover:bg-background/50">
                <td className="px-5 py-3.5 font-mono text-[11px] text-muted">{item.sku}</td>
                <td className="px-5 py-3.5 font-medium text-ink">{item.name}</td>
                <td className="px-5 py-3.5 text-xs text-muted">{item.category || '—'}</td>
                <td className="px-5 py-3.5 text-right font-mono text-xs text-ink">{formatNumber(item.stock)}</td>
                <td className="px-5 py-3.5 text-right font-mono text-xs text-muted">{formatNumber(item.min_stock)}</td>
                <td className="px-5 py-3.5 text-right font-mono text-xs text-ink">{formatCurrency(item.cost_price)}</td>
                <td className="px-5 py-3.5 text-right font-mono text-xs font-semibold text-ink">{formatCurrency(item.stock_value)}</td>
                <td className="px-5 py-3.5">
                  {item.is_low ? (
                    <Badge variant="warning" className="gap-1">
                      <AlertTriangle className="h-3 w-3" aria-hidden="true" />
                      Rendah
                    </Badge>
                  ) : (
                    <Badge variant="success">Aman</Badge>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </TableShell>
      )}
    </div>
  );
}

export function ValuationSection({ filter }: { filter: ReportFilter }) {
  const { data, loading, error } = useReportLoader(() => getInventoryValuation(filter), [filter]);

  if (loading) return <ReportSkeleton />;
  if (error) return <Alert variant="error">{error}</Alert>;
  if (!data) return null;

  return (
    <div className="space-y-4">
      <div className="grid gap-4 sm:grid-cols-3">
        <StatCard label="Total Nilai Inventori" value={formatCurrency(data.total_value)} hint="harga pokok × stok saat ini" />
        <StatCard label="Item Terlacak" value={formatNumber(data.total_items)} />
        <StatCard label="Stok Rendah" value={formatNumber(data.low_stock_rows)} />
      </div>

      {data.items.length === 0 ? (
        <EmptyState
          icon={TrendingUp}
          title="Belum ada inventori untuk dinilai"
          description="Nilai inventori dihitung dari stok sparepart saat ini."
          className="py-12"
        />
      ) : (
        <TableShell>
          <TableHead cells={['SKU', 'Sparepart', 'Qty', 'HPP Satuan', 'Nilai Stok']} />
          <tbody className="divide-y divide-border">
            {data.items.map((item) => (
              <tr key={item.sku} className="transition-colors hover:bg-background/50">
                <td className="px-5 py-3.5 font-mono text-[11px] text-muted">{item.sku}</td>
                <td className="px-5 py-3.5 font-medium text-ink">{item.name}</td>
                <td className="px-5 py-3.5 text-right font-mono text-xs text-ink">{formatNumber(item.stock)}</td>
                <td className="px-5 py-3.5 text-right font-mono text-xs text-ink">{formatCurrency(item.cost_price)}</td>
                <td className="px-5 py-3.5 text-right font-mono text-xs font-semibold text-ink">{formatCurrency(item.stock_value)}</td>
              </tr>
            ))}
            <tr className="border-t border-border bg-background font-semibold">
              <td className="px-5 py-3 text-xs uppercase tracking-wide text-muted" colSpan={4}>
                Total Nilai
              </td>
              <td className="px-5 py-3 text-right font-mono text-xs text-ink">{formatCurrency(data.total_value)}</td>
            </tr>
          </tbody>
        </TableShell>
      )}
    </div>
  );
}
