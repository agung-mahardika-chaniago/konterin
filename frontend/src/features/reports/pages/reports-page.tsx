import * as React from 'react';
import { Download, RotateCcw } from 'lucide-react';

import { Alert } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Select } from '@/components/ui/select';
import { rangeFor } from '@/features/dashboard/format';
import { hasPermission } from '@/lib/permissions';
import { cn } from '@/lib/utils';
import { useAuthStore } from '@/stores/auth-store';
import { exportReportXlsx, type ReportTab } from '../api';
import {
  BrandsSection,
  ExpensesSection,
  ProfitSection,
  RevenueSection,
  ServiceVsPartsSection,
  ServicesSection,
  StockSection,
  TechniciansSection,
  ValuationSection,
} from '../components/report-sections';
import type { ReportFilter, ReportPeriod } from '../types';

interface TabConfig {
  key: ReportTab;
  label: string;
  profitOnly?: boolean;
}

const TABS: TabConfig[] = [
  { key: 'revenue', label: 'Pendapatan' },
  { key: 'services', label: 'Servis' },
  { key: 'brands', label: 'Merek' },
  { key: 'service-vs-parts', label: 'Jasa vs Part' },
  { key: 'profit', label: 'Laba', profitOnly: true },
  { key: 'expenses', label: 'Pengeluaran', profitOnly: true },
  { key: 'technicians', label: 'Teknisi' },
  { key: 'stock', label: 'Stok' },
  { key: 'valuation', label: 'Nilai Inventori', profitOnly: true },
];

/** Valuation has no xlsx export on the server. */
const EXPORTABLE_TABS: ReportTab[] = [
  'revenue',
  'services',
  'brands',
  'service-vs-parts',
  'profit',
  'expenses',
  'technicians',
  'stock',
];

export function ReportsPage() {
  const user = useAuthStore((state) => state.user);
  const canViewProfit = hasPermission(user, 'report.profit.view');

  const defaultRange = React.useMemo(() => rangeFor(30), []);
  const [activeTab, setActiveTab] = React.useState<ReportTab>('revenue');
  const [outletId, setOutletId] = React.useState('');
  const [from, setFrom] = React.useState(defaultRange.from);
  const [to, setTo] = React.useState(defaultRange.to);
  const [period, setPeriod] = React.useState<ReportPeriod>('daily');
  const [exporting, setExporting] = React.useState(false);
  const [exportError, setExportError] = React.useState<string | null>(null);

  const visibleTabs = TABS.filter((tab) => !tab.profitOnly || canViewProfit);
  const effectiveTab: ReportTab = visibleTabs.some((tab) => tab.key === activeTab) ? activeTab : 'revenue';

  const filter = React.useMemo<ReportFilter>(
    () => ({ outlet_id: outletId || undefined, from, to }),
    [outletId, from, to],
  );

  const canExport = EXPORTABLE_TABS.includes(effectiveTab);

  const handleExport = async () => {
    setExporting(true);
    setExportError(null);
    try {
      await exportReportXlsx(effectiveTab, filter);
    } catch (err: unknown) {
      setExportError(err instanceof Error ? err.message : 'Gagal mengunduh berkas laporan.');
    } finally {
      setExporting(false);
    }
  };

  const resetFilters = () => {
    setOutletId('');
    setFrom(defaultRange.from);
    setTo(defaultRange.to);
    setPeriod('daily');
  };

  const isDefaultRange = outletId === '' && from === defaultRange.from && to === defaultRange.to && period === 'daily';

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-ink sm:text-[28px]">Laporan</h1>
          <p className="mt-1 text-sm text-muted">
            Ringkasan performa bisnis outlet: pendapatan, servis, laba, hingga nilai stok.
          </p>
        </div>
        <div className="flex flex-wrap gap-2 self-start sm:self-auto">
          {canExport && (
            <Button onClick={handleExport} loading={exporting} className="gap-2">
              <Download className="h-4 w-4" aria-hidden="true" />
              Ekspor Excel
            </Button>
          )}
        </div>
      </div>

      {exportError && <Alert variant="error">{exportError}</Alert>}

      <Card className="flex flex-col gap-3 p-4 lg:flex-row lg:items-center lg:justify-between lg:gap-4">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
          {user && user.outlets.length > 1 && (
            <Select
              value={outletId}
              onChange={(event) => setOutletId(event.target.value)}
              className="sm:max-w-[220px]"
              title="Outlet laporan"
            >
              <option value="">Outlet aktif ({user.outlets.find((o) => o.id === outletId)?.name ?? 'pilih'})</option>
              {user.outlets.map((outlet) => (
                <option key={outlet.id} value={outlet.id}>
                  {outlet.name}
                </option>
              ))}
            </Select>
          )}
          <div className="flex items-center gap-2">
            <Input
              type="date"
              value={from}
              onChange={(event) => setFrom(event.target.value)}
              className="w-[145px]"
              title="Tanggal awal"
            />
            <span className="text-xs text-muted">s.d.</span>
            <Input
              type="date"
              value={to}
              onChange={(event) => setTo(event.target.value)}
              className="w-[145px]"
              title="Tanggal akhir"
            />
          </div>
          {effectiveTab === 'revenue' && (
            <Select
              value={period}
              onChange={(event) => setPeriod(event.target.value as ReportPeriod)}
              className="sm:max-w-[160px]"
              title="Grup periode pendapatan"
            >
              <option value="daily">Harian</option>
              <option value="monthly">Bulanan</option>
              <option value="yearly">Tahunan</option>
            </Select>
          )}
        </div>
        {!isDefaultRange && (
          <Button variant="ghost" size="sm" onClick={resetFilters} className="gap-1.5 self-start">
            <RotateCcw className="h-3.5 w-3.5" aria-hidden="true" />
            Reset filter
          </Button>
        )}
      </Card>

      <div
        className="flex flex-wrap gap-1 rounded-control border border-border bg-surface p-1"
        role="tablist"
        aria-label="Jenis laporan"
      >
        {visibleTabs.map((tab) => (
          <button
            key={tab.key}
            type="button"
            role="tab"
            aria-selected={effectiveTab === tab.key}
            onClick={() => setActiveTab(tab.key)}
            className={cn(
              'rounded-control px-3 py-1.5 text-sm font-medium transition-colors',
              'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-500',
              effectiveTab === tab.key ? 'bg-brand-900 text-white shadow-sm' : 'text-muted hover:bg-background hover:text-ink',
            )}
          >
            {tab.label}
          </button>
        ))}
      </div>

      <div role="tabpanel" aria-label={TABS.find((tab) => tab.key === effectiveTab)?.label}>
        {effectiveTab === 'revenue' && <RevenueSection filter={filter} period={period} />}
        {effectiveTab === 'services' && <ServicesSection filter={filter} />}
        {effectiveTab === 'brands' && <BrandsSection filter={filter} />}
        {effectiveTab === 'service-vs-parts' && <ServiceVsPartsSection filter={filter} />}
        {effectiveTab === 'profit' && <ProfitSection filter={filter} />}
        {effectiveTab === 'expenses' && <ExpensesSection filter={filter} />}
        {effectiveTab === 'technicians' && <TechniciansSection filter={filter} />}
        {effectiveTab === 'stock' && <StockSection filter={filter} />}
        {effectiveTab === 'valuation' && <ValuationSection filter={filter} />}
      </div>
    </div>
  );
}
