import { AlertTriangle, CheckCircle2, Clock3, PackageX, Smartphone } from 'lucide-react';

import { EmptyState } from '@/components/shared/empty-state';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';

import { formatNumber } from './format';
import type { DashboardAlertItem, DashboardAlerts, DashboardStockAlert } from './types';

const statusLabels: Record<string, string> = {
  received: 'Diterima',
  waiting_diagnosis: 'Menunggu diagnosis',
  waiting_approval: 'Menunggu persetujuan',
  waiting_part: 'Menunggu sparepart',
  in_progress: 'Dikerjakan',
  testing: 'Pengujian',
  completed: 'Selesai',
  ready_for_pickup: 'Siap diambil',
  picked_up: 'Sudah diambil',
  cancelled: 'Dibatalkan',
};

function daysLabel(days: number): string {
  if (days <= 0) return 'Hari ini';
  return `${formatNumber(days)} hari`;
}

function AlertRow({ item, tone }: { item: DashboardAlertItem; tone: 'danger' | 'warning' }) {
  return (
    <li className="flex items-start gap-3 px-5 py-3">
      <span
        className={
          tone === 'danger'
            ? 'mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-red-50 text-danger'
            : 'mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-amber-50 text-warning'
        }
      >
        {tone === 'danger' ? (
          <AlertTriangle className="h-4 w-4" aria-hidden="true" />
        ) : (
          <Clock3 className="h-4 w-4" aria-hidden="true" />
        )}
      </span>
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-2">
          <span className="font-mono text-[11px] font-semibold text-brand-700">{item.ticket_no}</span>
          <Badge variant={tone === 'danger' ? 'danger' : 'warning'}>{daysLabel(item.days)}</Badge>
          <span className="text-[11px] text-muted">{statusLabels[item.status] ?? item.status}</span>
        </div>
        <p className="mt-0.5 truncate text-sm font-medium text-ink">{item.customer_name}</p>
        <p className="truncate text-xs text-muted">
          <Smartphone className="mr-1 inline h-3 w-3" aria-hidden="true" />
          {item.device}
        </p>
      </div>
    </li>
  );
}

function StockRow({ item }: { item: DashboardStockAlert }) {
  return (
    <li className="flex items-center gap-3 px-5 py-3">
      <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-violet-50 text-violet">
        <PackageX className="h-4 w-4" aria-hidden="true" />
      </span>
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium text-ink">{item.name}</p>
        <p className="truncate font-mono text-[11px] text-muted">{item.sku}</p>
      </div>
      <Badge variant="violet">
        {formatNumber(item.stock)} / min {formatNumber(item.min_stock)}
      </Badge>
    </li>
  );
}

export function AlertsPanel({ alerts, totalCustomers }: { alerts: DashboardAlerts; totalCustomers: number }) {
  const total = alerts.overdue.length + alerts.not_picked_up.length + alerts.low_stock.length;
  const isEmpty = total === 0;

  return (
    <Card id="alerts-panel" className="scroll-mt-24 overflow-hidden">
      <CardHeader className="flex-row items-start justify-between space-y-0 pb-3">
        <div>
          <CardTitle className="text-base">Perlu tindakan</CardTitle>
          <CardDescription>
            {formatNumber(totalCustomers)} pelanggan terdaftar di toko ini.
          </CardDescription>
        </div>
        <Badge variant={total > 0 ? 'danger' : 'success'}>{total > 0 ? `${total} item` : 'Aman'}</Badge>
      </CardHeader>
      <CardContent className="p-0">
        {isEmpty ? (
          <EmptyState
            icon={CheckCircle2}
            title="Tidak ada tindakan mendesak"
            description="Semua servis berjalan sesuai estimasi, perangkat sudah diambil, dan stok aman."
          />
        ) : (
          <div className="max-h-[430px] divide-y divide-border overflow-y-auto">
            {alerts.overdue.length > 0 && (
              <section aria-label="Servis lewat estimasi">
                <p className="bg-background/60 px-5 py-2 text-[11px] font-semibold uppercase tracking-wide text-muted">
                  Lewat estimasi ({alerts.overdue.length})
                </p>
                <ul className="divide-y divide-border/70">
                  {alerts.overdue.map((item) => (
                    <AlertRow key={item.id} item={item} tone="danger" />
                  ))}
                </ul>
              </section>
            )}

            {alerts.not_picked_up.length > 0 && (
              <section aria-label="Perangkat belum diambil">
                <p className="bg-background/60 px-5 py-2 text-[11px] font-semibold uppercase tracking-wide text-muted">
                  Belum diambil ({alerts.not_picked_up.length})
                </p>
                <ul className="divide-y divide-border/70">
                  {alerts.not_picked_up.map((item) => (
                    <AlertRow key={item.id} item={item} tone="warning" />
                  ))}
                </ul>
              </section>
            )}

            {alerts.low_stock.length > 0 && (
              <section aria-label="Stok sparepart menipis">
                <p className="bg-background/60 px-5 py-2 text-[11px] font-semibold uppercase tracking-wide text-muted">
                  Stok menipis ({alerts.low_stock.length})
                </p>
                <ul className="divide-y divide-border/70">
                  {alerts.low_stock.map((item) => (
                    <StockRow key={item.id} item={item} />
                  ))}
                </ul>
              </section>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
