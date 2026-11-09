import * as React from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { ChevronLeft, ChevronRight, Eye, RotateCcw, Search, ShieldCheck } from 'lucide-react';

import { EmptyState } from '@/components/shared/empty-state';
import { Alert } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Select } from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { useOutletStore } from '@/stores/outlet-store';
import { listClaims, listWarranties } from '../api';
import { ClaimStatusBadge, WarrantyStatusBadge, WarrantyTypeBadge } from '../components/status-badges';
import { formatDate, formatDateTime } from '../format';
import type { Claim, ClaimFilter, Warranty, WarrantyFilter } from '../types';

type TabKey = 'warranties' | 'claims';

export function WarrantyListPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const outletId = useOutletStore((state) => state.outletId);
  const tab: TabKey = searchParams.get('tab') === 'claims' ? 'claims' : 'warranties';

  const [search, setSearch] = React.useState(searchParams.get('q') ?? '');
  const [debouncedSearch, setDebouncedSearch] = React.useState(searchParams.get('q') ?? '');
  const [status, setStatus] = React.useState(searchParams.get('status') ?? '');
  const [expiring, setExpiring] = React.useState(searchParams.get('expiring_in_days') ?? '');
  const [page, setPage] = React.useState(1);

  const [warranties, setWarranties] = React.useState<Warranty[]>([]);
  const [claims, setClaims] = React.useState<Claim[]>([]);
  const [total, setTotal] = React.useState(0);
  const [totalPages, setTotalPages] = React.useState(1);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);

  React.useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search);
      setPage(1);
    }, 300);
    return () => clearTimeout(timer);
  }, [search]);

  const switchTab = (next: TabKey) => {
    setStatus('');
    setExpiring('');
    setPage(1);
    const params = new URLSearchParams(searchParams);
    if (next === 'claims') params.set('tab', 'claims');
    else params.delete('tab');
    setSearchParams(params, { replace: true });
  };

  const loadData = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      if (tab === 'warranties') {
        const filter: WarrantyFilter = {
          q: debouncedSearch.trim() || undefined,
          status: status || undefined,
          outlet_id: outletId ?? undefined,
          expiring_in_days: expiring ? Number(expiring) : undefined,
          page,
          per_page: 15,
        };
        const result = await listWarranties(filter);
        setWarranties(result.data ?? []);
        setTotal(result.meta?.total ?? 0);
        setTotalPages(result.meta?.total_pages ?? 1);
      } else {
        const filter: ClaimFilter = {
          q: debouncedSearch.trim() || undefined,
          status: status || undefined,
          outlet_id: outletId ?? undefined,
          page,
          per_page: 15,
        };
        const result = await listClaims(filter);
        setClaims(result.data ?? []);
        setTotal(result.meta?.total ?? 0);
        setTotalPages(result.meta?.total_pages ?? 1);
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Gagal memuat data garansi.');
    } finally {
      setLoading(false);
    }
  }, [tab, debouncedSearch, status, expiring, outletId, page]);

  React.useEffect(() => {
    loadData();
  }, [loadData]);

  const statusOptions =
    tab === 'warranties'
      ? [
          { value: 'active', label: 'Aktif' },
          { value: 'expired', label: 'Berakhir' },
          { value: 'void', label: 'Dibatalkan' },
        ]
      : [
          { value: 'submitted', label: 'Menunggu Review' },
          { value: 'approved', label: 'Disetujui' },
          { value: 'in_progress', label: 'Dikerjakan Ulang' },
          { value: 'completed', label: 'Selesai' },
          { value: 'rejected', label: 'Ditolak' },
        ];

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-ink">Garansi & Klaim</h1>
          <p className="mt-1 text-sm text-muted">
            Garansi dibuat otomatis saat perangkat diambil. Klaim tertaut ke servis asal dan dapat menjadi order rework.
          </p>
        </div>
        <Link to="/services" className="sm:self-start">
          <Button variant="outline" className="gap-2">
            <RotateCcw className="h-4 w-4" aria-hidden="true" />
            Daftar Servis
          </Button>
        </Link>
      </div>

      <div className="inline-flex rounded-control border border-border bg-surface p-0.5">
        <button
          type="button"
          onClick={() => switchTab('warranties')}
          className={`rounded px-3 py-1.5 text-xs font-semibold transition-colors ${
            tab === 'warranties' ? 'bg-brand-50 text-brand-900' : 'text-muted hover:text-ink'
          }`}
        >
          Garansi
        </button>
        <button
          type="button"
          onClick={() => switchTab('claims')}
          className={`rounded px-3 py-1.5 text-xs font-semibold transition-colors ${
            tab === 'claims' ? 'bg-brand-50 text-brand-900' : 'text-muted hover:text-ink'
          }`}
        >
          Klaim
        </button>
      </div>

      {error && <Alert variant="error">{error}</Alert>}

      <Card>
        <CardContent className="space-y-4 p-4 sm:p-5">
          <div className="grid gap-3 sm:grid-cols-[1fr_180px_160px]">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted" aria-hidden="true" />
              <Input
                value={search}
                onChange={(event) => setSearch(event.target.value)}
                placeholder={
                  tab === 'warranties'
                    ? 'Cari nomor garansi, tiket, pelanggan, atau perangkat...'
                    : 'Cari nomor klaim, garansi, tiket, atau pelanggan...'
                }
                className="pl-9"
                aria-label="Cari garansi atau klaim"
              />
            </div>
            <Select
              value={status}
              onChange={(event) => {
                setStatus(event.target.value);
                setPage(1);
              }}
              aria-label="Filter status"
            >
              <option value="">Semua status</option>
              {statusOptions.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </Select>
            {tab === 'warranties' ? (
              <Select
                value={expiring}
                onChange={(event) => {
                  setExpiring(event.target.value);
                  setPage(1);
                }}
                aria-label="Filter masa berlaku"
              >
                <option value="">Semua masa</option>
                <option value="7">Berakhir ≤ 7 hari</option>
                <option value="30">Berakhir ≤ 30 hari</option>
                <option value="60">Berakhir ≤ 60 hari</option>
              </Select>
            ) : (
              <div className="hidden sm:block" />
            )}
          </div>

          {loading ? (
            <div className="space-y-2">
              {[0, 1, 2, 3].map((item) => (
                <Skeleton key={item} className="h-16 w-full" />
              ))}
            </div>
          ) : tab === 'warranties' ? (
            warranties.length === 0 ? (
              <EmptyState
                icon={ShieldCheck}
                title="Belum ada garansi"
                description="Garansi terbentuk otomatis setelah servis berstatus diambil. Pastikan item servis memiliki durasi garansi."
              />
            ) : (
              <div className="divide-y divide-border">
                {warranties.map((warranty) => (
                  <div key={warranty.id} className="flex flex-col gap-2 py-3 sm:flex-row sm:items-center sm:justify-between">
                    <div className="min-w-0 space-y-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <Link
                          to={`/warranties/${warranty.id}`}
                          className="font-mono text-sm font-semibold text-accent-700 hover:underline"
                        >
                          {warranty.warranty_no}
                        </Link>
                        <WarrantyStatusBadge status={warranty.status} />
                        <WarrantyTypeBadge type={warranty.type} />
                      </div>
                      <p className="truncate text-sm text-ink">
                        {warranty.customer.name} · {warranty.device.brand} {warranty.device.model}
                      </p>
                      <p className="text-xs text-muted">
                        Tiket {warranty.service_order.ticket_no} · {formatDate(warranty.start_date)} –{' '}
                        {formatDate(warranty.end_date)} · {warranty.days} hari
                        {warranty.claim_count > 0 ? ` · ${warranty.claim_count} klaim` : ''}
                      </p>
                    </div>
                    <Link to={`/warranties/${warranty.id}`} className="shrink-0">
                      <Button variant="outline" size="sm" className="gap-1.5">
                        <Eye className="h-3.5 w-3.5" aria-hidden="true" />
                        Detail
                      </Button>
                    </Link>
                  </div>
                ))}
              </div>
            )
          ) : claims.length === 0 ? (
            <EmptyState
              icon={ShieldCheck}
              title="Belum ada klaim garansi"
              description="Klaim dapat dibuat dari halaman detail garansi yang masih aktif."
            />
          ) : (
            <div className="divide-y divide-border">
              {claims.map((claim) => (
                <div key={claim.id} className="flex flex-col gap-2 py-3 sm:flex-row sm:items-center sm:justify-between">
                  <div className="min-w-0 space-y-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <Link
                        to={`/claims/${claim.id}`}
                        className="font-mono text-sm font-semibold text-accent-700 hover:underline"
                      >
                        {claim.claim_no}
                      </Link>
                      <ClaimStatusBadge status={claim.status} />
                    </div>
                    <p className="truncate text-sm text-ink">{claim.complaint}</p>
                    <p className="text-xs text-muted">
                      {claim.customer.name} · {claim.device.brand} {claim.device.model} · Tiket{' '}
                      {claim.service_order.ticket_no} · {formatDateTime(claim.claimed_at)}
                    </p>
                  </div>
                  <div className="flex shrink-0 items-center gap-2">
                    {claim.rework_service_order && (
                      <span className="text-[11px] text-muted">Rework {claim.rework_service_order.ticket_no}</span>
                    )}
                    <Link to={`/claims/${claim.id}`}>
                      <Button variant="outline" size="sm" className="gap-1.5">
                        <Eye className="h-3.5 w-3.5" aria-hidden="true" />
                        Detail
                      </Button>
                    </Link>
                  </div>
                </div>
              ))}
            </div>
          )}

          {!loading && totalPages > 1 && (
            <div className="flex items-center justify-between border-t border-border pt-3">
              <p className="text-xs text-muted">
                {total} data · halaman {page} dari {totalPages}
              </p>
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page <= 1}
                  onClick={() => setPage((value) => Math.max(1, value - 1))}
                  className="gap-1"
                >
                  <ChevronLeft className="h-3.5 w-3.5" aria-hidden="true" />
                  Sebelumnya
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page >= totalPages}
                  onClick={() => setPage((value) => Math.min(totalPages, value + 1))}
                  className="gap-1"
                >
                  Berikutnya
                  <ChevronRight className="h-3.5 w-3.5" aria-hidden="true" />
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
