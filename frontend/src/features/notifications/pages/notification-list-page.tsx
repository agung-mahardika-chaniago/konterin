import * as React from 'react';
import { useNavigate } from 'react-router-dom';
import { Bell, CheckCheck, ChevronLeft, ChevronRight } from 'lucide-react';

import { EmptyState } from '@/components/shared/empty-state';
import { Alert } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';
import {
  getUnreadCount,
  listNotifications,
  markAllNotificationsRead,
  markNotificationRead,
} from '../api';
import { NotificationIcon } from '../components/notification-icon';
import { formatDateTime, timeAgo } from '../format';
import type { AppNotification } from '../types';

export function NotificationListPage() {
  const navigate = useNavigate();
  const [unreadOnly, setUnreadOnly] = React.useState(false);
  const [items, setItems] = React.useState<AppNotification[]>([]);
  const [unread, setUnread] = React.useState(0);
  const [page, setPage] = React.useState(1);
  const [total, setTotal] = React.useState(0);
  const [totalPages, setTotalPages] = React.useState(1);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);

  const load = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [result, count] = await Promise.all([
        listNotifications({ unread: unreadOnly, page, per_page: 20 }),
        getUnreadCount(),
      ]);
      setItems(result.data ?? []);
      setTotal(result.meta?.total ?? 0);
      setTotalPages(result.meta?.total_pages ?? 1);
      setUnread(count);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Gagal memuat notifikasi.');
    } finally {
      setLoading(false);
    }
  }, [unreadOnly, page]);

  React.useEffect(() => {
    load();
  }, [load]);

  const openNotification = async (item: AppNotification) => {
    if (!item.is_read) {
      setUnread((value) => Math.max(0, value - 1));
      setItems((previous) =>
        previous.map((entry) => (entry.id === item.id ? { ...entry, is_read: true } : entry)),
      );
      try {
        await markNotificationRead(item.id);
      } catch {
        // The next load reconciles read state.
      }
    }
    navigate(item.link || '/notifications');
  };

  const markAllRead = async () => {
    setError(null);
    try {
      await markAllNotificationsRead();
      setUnread(0);
      setItems((previous) => previous.map((entry) => ({ ...entry, is_read: true })));
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Gagal menandai notifikasi.');
    }
  };

  const switchFilter = (next: boolean) => {
    setUnreadOnly(next);
    setPage(1);
  };

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-ink">Notifikasi</h1>
          <p className="mt-1 text-sm text-muted">
            Pemberitahuan servis siap diambil, klaim garansi, pengingat, dan aktivitas operasional lain.
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={markAllRead}
          disabled={unread === 0}
          className="gap-2 self-start"
        >
          <CheckCheck className="h-4 w-4" aria-hidden="true" />
          Tandai Semua Dibaca
        </Button>
      </div>

      <div className="inline-flex rounded-control border border-border bg-surface p-0.5">
        <button
          type="button"
          onClick={() => switchFilter(false)}
          className={cn(
            'rounded px-3 py-1.5 text-xs font-semibold transition-colors',
            !unreadOnly ? 'bg-brand-50 text-brand-900' : 'text-muted hover:text-ink',
          )}
        >
          Semua
        </button>
        <button
          type="button"
          onClick={() => switchFilter(true)}
          className={cn(
            'rounded px-3 py-1.5 text-xs font-semibold transition-colors',
            unreadOnly ? 'bg-brand-50 text-brand-900' : 'text-muted hover:text-ink',
          )}
        >
          Belum Dibaca{unread > 0 ? ` (${unread})` : ''}
        </button>
      </div>

      {error && <Alert variant="error">{error}</Alert>}

      <Card>
        <CardContent className="p-0">
          {loading ? (
            <div className="space-y-2 p-4">
              {[0, 1, 2, 3, 4].map((item) => (
                <Skeleton key={item} className="h-16 w-full" />
              ))}
            </div>
          ) : items.length === 0 ? (
            <EmptyState
              icon={Bell}
              title={unreadOnly ? 'Tidak ada notifikasi belum dibaca' : 'Belum ada notifikasi'}
              description={
                unreadOnly
                  ? 'Semua notifikasi sudah dibaca. Matikan filter untuk melihat riwayat.'
                  : 'Notifikasi akan muncul saat servis siap diambil, klaim garansi dibuat, atau pengingat worker berjalan.'
              }
            />
          ) : (
            <ul className="divide-y divide-border">
              {items.map((item) => (
                <li key={item.id}>
                  <button
                    type="button"
                    onClick={() => openNotification(item)}
                    className={cn(
                      'flex w-full items-start gap-3 px-4 py-4 text-left transition-colors hover:bg-background',
                      !item.is_read && 'bg-brand-50/50',
                    )}
                  >
                    <span
                      className={cn(
                        'mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-full',
                        item.is_read ? 'bg-slate-100 text-slate-600' : 'bg-brand-100 text-brand-800',
                      )}
                    >
                      <NotificationIcon type={item.type} className="h-4 w-4" />
                    </span>
                    <span className="min-w-0 flex-1">
                      <span className="flex flex-wrap items-center gap-2">
                        <span className="text-sm font-semibold text-ink">{item.title}</span>
                        {!item.is_read && (
                          <span className="rounded-full bg-accent-100 px-1.5 py-0.5 text-[9px] font-bold uppercase tracking-wide text-accent-800">
                            Baru
                          </span>
                        )}
                        {item.is_broadcast && (
                          <span className="rounded-full bg-slate-100 px-1.5 py-0.5 text-[9px] font-semibold uppercase tracking-wide text-slate-600">
                            Broadcast
                          </span>
                        )}
                      </span>
                      {item.body && <span className="mt-1 block text-xs text-muted">{item.body}</span>}
                      <span className="mt-1 block text-[11px] text-muted" title={formatDateTime(item.created_at)}>
                        {timeAgo(item.created_at)} · {formatDateTime(item.created_at)}
                      </span>
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          )}

          {!loading && totalPages > 1 && (
            <div className="flex items-center justify-between border-t border-border px-4 py-3">
              <p className="text-xs text-muted">
                {total} notifikasi · halaman {page} dari {totalPages}
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
