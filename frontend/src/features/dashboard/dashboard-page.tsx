import {
  CalendarDays,
  CheckCircle2,
  ClipboardList,
  Clock3,
  PackageOpen,
  RefreshCw,
  Store,
  TrendingUp,
  Wallet,
  Wrench,
} from 'lucide-react';
import { useState } from 'react';

import { EmptyState } from '@/components/shared/empty-state';
import { Alert } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { PanelSkeleton, Skeleton, StatCardSkeleton } from '@/components/ui/skeleton';
import { useAuthStore } from '@/stores/auth-store';
import { useOutletStore } from '@/stores/outlet-store';
import { cn } from '@/lib/utils';

import { AlertsPanel } from './alerts-panel';
import {
  formatCompactCurrency,
  formatCurrency,
  formatNumber,
  greetingFor,
  longToday,
} from './format';
import { RevenueChart } from './revenue-chart';
import { ServiceChart } from './service-chart';
import { StatCard } from './stat-card';
import { TechnicianPanel } from './technician-panel';
import type { DashboardRange } from './types';
import { useDashboardData } from './use-dashboard';

function ChartCardSkeleton() {
  return (
    <Card>
      <CardHeader className="pb-2">
        <Skeleton className="h-4 w-36" />
        <Skeleton className="h-3 w-48" />
      </CardHeader>
      <CardContent>
        <Skeleton className="h-[260px] w-full" />
      </CardContent>
    </Card>
  );
}

export function DashboardPage() {
  const user = useAuthStore((state) => state.user);
  const storedOutletId = useOutletStore((state) => state.outletId);
  const [range, setRange] = useState<DashboardRange>(7);

  const outletId = storedOutletId ?? user?.outlets[0]?.id ?? null;
  const activeOutlet = user?.outlets.find((outlet) => outlet.id === outletId);
  const { data, status, error, refreshing, refresh } = useDashboardData(outletId, range);

  if (!user) return null;

  const firstName = user.name.split(/\s+/)[0] ?? user.name;
  const totalRevenue = data?.charts.days.reduce((sum, day) => sum + day.revenue, 0) ?? 0;

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <p className="text-[11px] font-semibold uppercase tracking-[0.14em] text-accent-600">
              {activeOutlet?.name ?? 'Outlet aktif'}
            </p>
            {activeOutlet && <Badge variant="neutral">{activeOutlet.code}</Badge>}
          </div>
          <h1 className="mt-1 text-2xl font-bold tracking-tight text-ink sm:text-[28px]">
            {greetingFor(new Date())}, {firstName} 👋
          </h1>
          <p className="mt-1 flex items-center gap-1.5 text-sm text-muted">
            <CalendarDays className="h-4 w-4" aria-hidden="true" />
            Ringkasan {longToday().toLowerCase()}
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <div
            className="inline-flex rounded-control border border-border bg-surface p-0.5 shadow-sm"
            role="group"
            aria-label="Rentang grafik"
          >
            {([7, 30] as const).map((option) => (
              <button
                key={option}
                type="button"
                onClick={() => setRange(option)}
                aria-pressed={range === option}
                className={cn(
                  'min-h-[36px] rounded-[7px] px-3 text-xs font-semibold transition-colors',
                  range === option ? 'bg-brand-900 text-white shadow-sm' : 'text-muted hover:bg-background hover:text-ink',
                )}
              >
                {option} hari
              </button>
            ))}
          </div>
          <Button variant="outline" size="sm" onClick={refresh} disabled={refreshing || status === 'loading'}>
            <RefreshCw className={cn('h-4 w-4', refreshing && 'animate-spin')} aria-hidden="true" />
            {refreshing ? 'Memuat…' : 'Muat ulang'}
          </Button>
        </div>
      </div>

      {error && (
        <Alert variant="error">
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <span>{error}</span>
            <Button variant="outline" size="sm" onClick={refresh} className="self-start bg-white">
              Coba lagi
            </Button>
          </div>
        </Alert>
      )}

      {!outletId ? (
        <Card>
          <EmptyState
            icon={Store}
            title="Belum ada outlet aktif"
            description="Minta owner atau admin menugaskan Anda ke outlet sebelum membuka dashboard."
          />
        </Card>
      ) : status === 'loading' ? (
        <>
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 xl:grid-cols-4">
            {Array.from({ length: 7 }).map((_, index) => (
              <StatCardSkeleton key={index} />
            ))}
          </div>
          <div className="grid gap-4 xl:grid-cols-2">
            <ChartCardSkeleton />
            <ChartCardSkeleton />
          </div>
          <div className="grid gap-4 xl:grid-cols-2">
            <Card>
              <CardHeader>
                <Skeleton className="h-4 w-32" />
                <Skeleton className="h-3 w-48" />
              </CardHeader>
              <PanelSkeleton />
            </Card>
            <Card>
              <CardHeader>
                <Skeleton className="h-4 w-32" />
                <Skeleton className="h-3 w-40" />
              </CardHeader>
              <PanelSkeleton />
            </Card>
          </div>
        </>
      ) : status === 'error' || !data ? (
        <Card>
          <EmptyState
            icon={RefreshCw}
            title="Dashboard gagal dimuat"
            description="Periksa koneksi server lalu muat ulang data."
            action={
              <Button size="sm" onClick={refresh}>
                Coba lagi
              </Button>
            }
          />
        </Card>
      ) : (
        <>
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 xl:grid-cols-4">
            <StatCard
              icon={ClipboardList}
              label="Servis hari ini"
              value={formatNumber(data.summary.services_today)}
              hint="Tiket masuk hari ini"
              tone="brand"
            />
            <StatCard
              icon={Clock3}
              label="Menunggu pengerjaan"
              value={formatNumber(data.summary.waiting_count)}
              hint="Termasuk menunggu sparepart"
              tone="warning"
            />
            <StatCard
              icon={Wrench}
              label="Dikerjakan"
              value={formatNumber(data.summary.in_progress_count)}
              hint="Sedang dikerjakan / diuji"
              tone="info"
            />
            <StatCard
              icon={CheckCircle2}
              label="Selesai hari ini"
              value={formatNumber(data.summary.completed_count)}
              hint="Servis selesai hari ini"
              tone="success"
            />
            <StatCard
              icon={PackageOpen}
              label="Belum diambil"
              value={formatNumber(data.summary.not_picked_up_count)}
              hint="Selesai / siap diambil"
              tone="danger"
            />
            <StatCard
              icon={Wallet}
              label="Pendapatan hari ini"
              value={formatCurrency(data.summary.revenue_today)}
              hint={`${formatCompactCurrency(data.summary.revenue_today)} diterima`}
              tone="accent"
            />
            <StatCard
              icon={TrendingUp}
              label="Pendapatan bulan ini"
              value={formatCurrency(data.summary.revenue_month)}
              hint={`${formatCompactCurrency(data.summary.revenue_month)} bulan berjalan`}
              tone="violet"
            />
          </div>

          <div className="grid gap-4 xl:grid-cols-2">
            <Card>
              <CardHeader className="flex-row items-start justify-between space-y-0 pb-2">
                <div>
                  <CardTitle className="text-base">Pendapatan</CardTitle>
                  <CardDescription>{range} hari terakhir · total {formatCurrency(totalRevenue)}</CardDescription>
                </div>
                <Badge variant="brand">Posted</Badge>
              </CardHeader>
              <CardContent>
                <RevenueChart days={data.charts.days} />
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex-row items-start justify-between space-y-0 pb-2">
                <div>
                  <CardTitle className="text-base">Servis</CardTitle>
                  <CardDescription>Masuk dan selesai per hari</CardDescription>
                </div>
                <div className="flex items-center gap-3 text-[11px] text-muted">
                  <span className="flex items-center gap-1.5">
                    <span className="h-2 w-2 rounded-full bg-brand-700" aria-hidden="true" />
                    Masuk
                  </span>
                  <span className="flex items-center gap-1.5">
                    <span className="h-2 w-2 rounded-full bg-accent-500" aria-hidden="true" />
                    Selesai
                  </span>
                </div>
              </CardHeader>
              <CardContent>
                <ServiceChart days={data.charts.days} />
              </CardContent>
            </Card>
          </div>

          <div className="grid gap-4 xl:grid-cols-2">
            <AlertsPanel alerts={data.alerts} totalCustomers={data.summary.total_customers} />
            <TechnicianPanel technicians={data.technicians} days={range} />
          </div>
        </>
      )}
    </div>
  );
}
