import * as React from 'react';
import {
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  ChevronUp,
  History,
  RotateCcw,
  Search,
} from 'lucide-react';

import { EmptyState } from '@/components/shared/empty-state';
import { Alert } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Select } from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { formatDateTime } from '@/features/invoices/format';
import { listAuditLogs, listAuditModules } from '../api';
import type { AuditLogRow } from '../types';

function prettyJson(value: unknown): string {
  if (value === null || value === undefined) return '—';
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}

function DetailRow({ row }: { row: AuditLogRow }) {
  return (
    <tr className="bg-background/60">
      <td colSpan={5} className="px-5 py-3">
        <div className="grid gap-3 lg:grid-cols-2">
          <div>
            <p className="mb-1 text-[11px] font-semibold uppercase tracking-wide text-muted">Sebelum</p>
            <pre className="max-h-56 overflow-auto rounded-control border border-border bg-surface p-3 font-mono text-[11px] leading-5 text-muted">
              {prettyJson(row.before_data)}
            </pre>
          </div>
          <div>
            <p className="mb-1 text-[11px] font-semibold uppercase tracking-wide text-muted">Sesudah</p>
            <pre className="max-h-56 overflow-auto rounded-control border border-border bg-surface p-3 font-mono text-[11px] leading-5 text-ink">
              {prettyJson(row.after_data)}
            </pre>
          </div>
        </div>
      </td>
    </tr>
  );
}

export function AuditLogPage() {
  const [items, setItems] = React.useState<AuditLogRow[]>([]);
  const [modules, setModules] = React.useState<string[]>([]);
  const [total, setTotal] = React.useState(0);
  const [page, setPage] = React.useState(1);
  const [totalPages, setTotalPages] = React.useState(1);
  const [action, setAction] = React.useState('');
  const [module, setModule] = React.useState('');
  const [entityType, setEntityType] = React.useState('');
  const [from, setFrom] = React.useState('');
  const [to, setTo] = React.useState('');
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);
  const [expanded, setExpanded] = React.useState<number | null>(null);

  React.useEffect(() => {
    listAuditModules()
      .then((result) => setModules(result ?? []))
      .catch(() => setModules([]));
  }, []);

  const loadData = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await listAuditLogs({
        action: action.trim() || undefined,
        module: module || undefined,
        entity_type: entityType.trim() || undefined,
        from: from || undefined,
        to: to || undefined,
        page,
        per_page: 20,
      });
      setItems(result.data?.items ?? []);
      setExpanded(null);
      if (result.meta) {
        setTotal(result.meta.total);
        setTotalPages(result.meta.total_pages || 1);
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Gagal memuat audit log.');
    } finally {
      setLoading(false);
    }
  }, [action, module, entityType, from, to, page]);

  React.useEffect(() => {
    loadData();
  }, [loadData]);

  const resetFilters = () => {
    setAction('');
    setModule('');
    setEntityType('');
    setFrom('');
    setTo('');
    setPage(1);
  };

  const hasFilter = Boolean(action || module || entityType || from || to);

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-ink sm:text-[28px]">Audit Log</h1>
          <p className="mt-1 text-sm text-muted">
            Jejak perubahan data toko — siapa mengubah apa, kapan, dan apa isinya sebelum serta sesudah.
          </p>
        </div>
      </div>

      {error && <Alert variant="error">{error}</Alert>}

      <div className="flex flex-col gap-3 xl:flex-row xl:items-center xl:flex-wrap">
        <div className="relative flex-1 xl:max-w-xs">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted" aria-hidden="true" />
          <Input
            value={action}
            onChange={(event) => {
              setAction(event.target.value);
              setPage(1);
            }}
            placeholder="Cari aksi (create, update...)"
            className="pl-9"
          />
        </div>
        <Select
          value={module}
          onChange={(event) => {
            setModule(event.target.value);
            setPage(1);
          }}
          className="xl:max-w-[180px]"
        >
          <option value="">Semua modul</option>
          {modules.map((value) => (
            <option key={value} value={value}>
              {value}
            </option>
          ))}
        </Select>
        <Input
          value={entityType}
          onChange={(event) => {
            setEntityType(event.target.value);
            setPage(1);
          }}
          placeholder="Tipe entitas (service_order...)"
          className="xl:max-w-[220px]"
        />
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
          <Button variant="ghost" size="sm" onClick={resetFilters} className="gap-1.5">
            <RotateCcw className="h-3.5 w-3.5" aria-hidden="true" />
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
            icon={History}
            title="Belum ada aktivitas"
            description={
              hasFilter
                ? 'Tidak ada audit log yang sesuai dengan filter Anda.'
                : 'Perubahan data akan tercatat di sini secara otomatis.'
            }
            className="py-12"
          />
        ) : (
          <>
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead className="border-b border-border bg-background text-xs uppercase tracking-wider text-muted">
                  <tr>
                    <th scope="col" className="px-5 py-3 font-semibold">Waktu</th>
                    <th scope="col" className="px-5 py-3 font-semibold">Aktor</th>
                    <th scope="col" className="px-5 py-3 font-semibold">Modul</th>
                    <th scope="col" className="px-5 py-3 font-semibold">Aksi &amp; Entitas</th>
                    <th scope="col" className="px-5 py-3 font-semibold" aria-label="Rincian" />
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {items.map((row) => (
                    <React.Fragment key={row.id}>
                      <tr
                        className={`transition-colors hover:bg-background/50 ${expanded === row.id ? 'bg-background/50' : ''}`}
                      >
                        <td className="whitespace-nowrap px-5 py-3.5 text-xs text-muted">
                          {formatDateTime(row.created_at)}
                        </td>
                        <td className="px-5 py-3.5 text-xs font-medium text-ink">{row.actor_name || 'Sistem'}</td>
                        <td className="px-5 py-3.5">
                          <Badge variant="neutral">{row.module}</Badge>
                        </td>
                        <td className="px-5 py-3.5 text-xs text-ink">
                          <span className="font-medium">{row.action}</span>
                          <span className="text-muted"> · {row.entity_type}</span>
                          {row.entity_id && (
                            <span className="ml-1 font-mono text-[11px] text-muted">#{row.entity_id.slice(0, 8)}</span>
                          )}
                        </td>
                        <td className="whitespace-nowrap px-5 py-3.5 text-right">
                          {(row.before_data !== null && row.before_data !== undefined) ||
                          (row.after_data !== null && row.after_data !== undefined) ? (
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => setExpanded(expanded === row.id ? null : row.id)}
                              className="h-8 gap-1 px-2 text-xs"
                              aria-expanded={expanded === row.id}
                            >
                              {expanded === row.id ? (
                                <>
                                  Tutup
                                  <ChevronUp className="h-3.5 w-3.5" aria-hidden="true" />
                                </>
                              ) : (
                                <>
                                  Rincian
                                  <ChevronDown className="h-3.5 w-3.5" aria-hidden="true" />
                                </>
                              )}
                            </Button>
                          ) : null}
                        </td>
                      </tr>
                      {expanded === row.id && <DetailRow row={row} />}
                    </React.Fragment>
                  ))}
                </tbody>
              </table>
            </div>

            <div className="flex items-center justify-between border-t border-border px-5 py-3 text-xs text-muted">
              <span>
                Menampilkan {items.length} dari total {total} entri
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
