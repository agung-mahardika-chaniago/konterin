import * as React from 'react';
import { Link } from 'react-router-dom';
import {
  AlertTriangle,
  ChevronLeft,
  ChevronRight,
  MessageCircle,
  RefreshCw,
  Search,
  Send,
} from 'lucide-react';

import { EmptyState } from '@/components/shared/empty-state';
import { Alert } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Select } from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { listWhatsAppMessages, retryWhatsAppMessage } from '../api';
import { formatDateTime, templateLabel, whatsappStatusLabel, whatsappStatusVariant } from '../format';
import type { WhatsAppMessage } from '../types';

export function WhatsAppOutboxPage() {
  const [search, setSearch] = React.useState('');
  const [debouncedSearch, setDebouncedSearch] = React.useState('');
  const [status, setStatus] = React.useState('');
  const [items, setItems] = React.useState<WhatsAppMessage[]>([]);
  const [page, setPage] = React.useState(1);
  const [total, setTotal] = React.useState(0);
  const [totalPages, setTotalPages] = React.useState(1);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);
  const [retrying, setRetrying] = React.useState<string | null>(null);

  React.useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search);
      setPage(1);
    }, 300);
    return () => clearTimeout(timer);
  }, [search]);

  const load = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await listWhatsAppMessages({
        q: debouncedSearch.trim() || undefined,
        status: status || undefined,
        page,
        per_page: 20,
      });
      setItems(result.data ?? []);
      setTotal(result.meta?.total ?? 0);
      setTotalPages(result.meta?.total_pages ?? 1);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Gagal memuat outbox WhatsApp.');
    } finally {
      setLoading(false);
    }
  }, [debouncedSearch, status, page]);

  React.useEffect(() => {
    load();
  }, [load]);

  const retry = async (item: WhatsAppMessage) => {
    setRetrying(item.id);
    setError(null);
    try {
      const updated = await retryWhatsAppMessage(item.id);
      setItems((previous) => previous.map((entry) => (entry.id === item.id ? updated : entry)));
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Gagal mengirim ulang pesan WhatsApp.');
    } finally {
      setRetrying(null);
    }
  };

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-ink">Outbox WhatsApp</h1>
          <p className="mt-1 text-sm text-muted">
            Riwayat pesan keluar, status pengiriman, error provider, dan kirim ulang pesan yang gagal.
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={load} disabled={loading} className="gap-2 self-start">
          <RefreshCw className={loading ? 'h-4 w-4 animate-spin' : 'h-4 w-4'} aria-hidden="true" />
          Muat Ulang
        </Button>
      </div>

      <Alert variant="info">
        Worker memproses outbox secara berkala. Provider `noop` hanya mencatat pesan tanpa mengirim; isi token
        Fonnte/Wablas di environment untuk pengiriman nyata.
      </Alert>

      {error && <Alert variant="error">{error}</Alert>}

      <Card>
        <CardContent className="space-y-4 p-4 sm:p-5">
          <div className="grid gap-3 sm:grid-cols-[1fr_190px]">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted" aria-hidden="true" />
              <Input
                value={search}
                onChange={(event) => setSearch(event.target.value)}
                placeholder="Cari nomor HP, tiket, atau nama pelanggan..."
                className="pl-9"
                aria-label="Cari pesan WhatsApp"
              />
            </div>
            <Select
              value={status}
              onChange={(event) => {
                setStatus(event.target.value);
                setPage(1);
              }}
              aria-label="Filter status pesan"
            >
              <option value="">Semua status</option>
              <option value="queued">Menunggu</option>
              <option value="processing">Diproses</option>
              <option value="sent">Terkirim</option>
              <option value="failed">Gagal</option>
              <option value="cancelled">Dibatalkan</option>
            </Select>
          </div>

          {loading ? (
            <div className="space-y-2">
              {[0, 1, 2, 3].map((item) => (
                <Skeleton key={item} className="h-24 w-full" />
              ))}
            </div>
          ) : items.length === 0 ? (
            <EmptyState
              icon={MessageCircle}
              title="Outbox masih kosong"
              description="Pesan otomatis akan masuk ke outbox saat servis siap diambil, klaim garansi dibuat, atau worker mengirim pengingat."
            />
          ) : (
            <ul className="divide-y divide-border">
              {items.map((item) => (
                <li key={item.id} className="py-4">
                  <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                    <div className="min-w-0 flex-1 space-y-1.5">
                      <div className="flex flex-wrap items-center gap-2">
                        <Badge variant={whatsappStatusVariant(item.status)}>{whatsappStatusLabel(item.status)}</Badge>
                        <span className="text-xs font-semibold text-ink">{templateLabel(item.template_key)}</span>
                        <span className="rounded-full bg-slate-100 px-1.5 py-0.5 text-[10px] font-semibold text-slate-600">
                          {item.provider}
                        </span>
                        {item.attempts > 0 && (
                          <span className="text-[10px] text-muted">{item.attempts} percobaan</span>
                        )}
                      </div>

                      <p className="text-sm font-medium text-ink">
                        {item.customer_name || 'Pelanggan'}{' '}
                        <span className="font-mono text-xs text-muted">· {item.phone}</span>
                      </p>

                      {item.body && <p className="line-clamp-2 text-xs text-muted">{item.body}</p>}

                      <p className="flex flex-wrap items-center gap-x-3 gap-y-1 text-[11px] text-muted">
                        <span>Dibuat {formatDateTime(item.created_at)}</span>
                        <span>Jadwal {formatDateTime(item.scheduled_at)}</span>
                        {item.sent_at && <span>Terkirim {formatDateTime(item.sent_at)}</span>}
                        {item.ticket_no && (
                          <span>
                            Tiket{' '}
                            {item.service_order_id ? (
                              <Link
                                to={`/services/${item.service_order_id}`}
                                className="font-mono font-semibold text-accent-700 hover:underline"
                              >
                                {item.ticket_no}
                              </Link>
                            ) : (
                              <span className="font-mono">{item.ticket_no}</span>
                            )}
                          </span>
                        )}
                      </p>

                      {item.last_error && (
                        <p className="flex items-start gap-1.5 rounded-control border border-red-200 bg-red-50 p-2 text-[11px] text-red-800">
                          <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" aria-hidden="true" />
                          {item.last_error}
                        </p>
                      )}
                    </div>

                    {item.status === 'failed' && (
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => retry(item)}
                        loading={retrying === item.id}
                        className="shrink-0 gap-1.5"
                      >
                        <Send className="h-3.5 w-3.5" aria-hidden="true" />
                        Kirim Ulang
                      </Button>
                    )}
                  </div>
                </li>
              ))}
            </ul>
          )}

          {!loading && totalPages > 1 && (
            <div className="flex items-center justify-between border-t border-border pt-3">
              <p className="text-xs text-muted">
                {total} pesan · halaman {page} dari {totalPages}
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
