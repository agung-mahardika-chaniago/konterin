import * as React from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Bell, CheckCheck, Loader2 } from 'lucide-react';

import { cn } from '@/lib/utils';
import {
  getUnreadCount,
  listNotifications,
  markAllNotificationsRead,
  markNotificationRead,
} from '../api';
import { timeAgo } from '../format';
import { NotificationIcon } from './notification-icon';
import type { AppNotification } from '../types';

export function NotificationMenu() {
  const navigate = useNavigate();
  const containerRef = React.useRef<HTMLDivElement>(null);
  const [open, setOpen] = React.useState(false);
  const [items, setItems] = React.useState<AppNotification[]>([]);
  const [unread, setUnread] = React.useState(0);
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  const loadUnread = React.useCallback(async () => {
    try {
      setUnread(await getUnreadCount());
    } catch {
      // The badge is non-critical; keep the previous count on transient errors.
    }
  }, []);

  const loadItems = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await listNotifications({ per_page: 6 });
      setItems(result.data ?? []);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Gagal memuat notifikasi.');
    } finally {
      setLoading(false);
    }
  }, []);

  React.useEffect(() => {
    loadUnread();
    const timer = window.setInterval(loadUnread, 60_000);
    return () => window.clearInterval(timer);
  }, [loadUnread]);

  React.useEffect(() => {
    if (!open) return;
    loadItems();
    loadUnread();
  }, [open, loadItems, loadUnread]);

  React.useEffect(() => {
    if (!open) return;
    const handlePointerDown = (event: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setOpen(false);
      }
    };
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setOpen(false);
    };
    document.addEventListener('mousedown', handlePointerDown);
    document.addEventListener('keydown', handleKeyDown);
    return () => {
      document.removeEventListener('mousedown', handlePointerDown);
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [open]);

  const openNotification = async (item: AppNotification) => {
    setOpen(false);
    if (!item.is_read) {
      setUnread((value) => Math.max(0, value - 1));
      setItems((previous) =>
        previous.map((entry) => (entry.id === item.id ? { ...entry, is_read: true } : entry)),
      );
      try {
        await markNotificationRead(item.id);
      } catch {
        // Navigation still works; the next poll reconciles the badge count.
      }
    }
    navigate(item.link || '/notifications');
  };

  const markAllRead = async () => {
    try {
      await markAllNotificationsRead();
      setUnread(0);
      setItems((previous) => previous.map((entry) => ({ ...entry, is_read: true })));
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Gagal menandai notifikasi.');
    }
  };

  return (
    <div ref={containerRef} className="relative shrink-0">
      <button
        type="button"
        onClick={() => setOpen((value) => !value)}
        className="relative flex h-10 w-10 items-center justify-center rounded-control text-ink hover:bg-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-500"
        aria-label={unread > 0 ? `${unread} notifikasi belum dibaca` : 'Notifikasi'}
        aria-haspopup="menu"
        aria-expanded={open}
      >
        <Bell className="h-5 w-5" aria-hidden="true" />
        {unread > 0 && (
          <span className="absolute right-1 top-1 flex min-w-[18px] items-center justify-center rounded-full bg-danger px-1 text-[10px] font-bold leading-[18px] text-white">
            {unread > 99 ? '99+' : unread}
          </span>
        )}
      </button>

      {open && (
        <div
          role="menu"
          aria-label="Notifikasi"
          className="absolute right-0 top-12 z-50 w-[340px] max-w-[calc(100vw-1.5rem)] overflow-hidden rounded-xl border border-border bg-surface shadow-xl"
        >
          <div className="flex items-center justify-between gap-2 border-b border-border px-4 py-3">
            <div>
              <p className="text-sm font-semibold text-ink">Notifikasi</p>
              <p className="text-[11px] text-muted">{unread > 0 ? `${unread} belum dibaca` : 'Semua sudah dibaca'}</p>
            </div>
            <button
              type="button"
              onClick={markAllRead}
              disabled={unread === 0}
              className="inline-flex items-center gap-1 rounded-control px-2 py-1 text-[11px] font-semibold text-accent-700 hover:bg-background disabled:cursor-not-allowed disabled:opacity-40"
            >
              <CheckCheck className="h-3.5 w-3.5" aria-hidden="true" />
              Tandai semua
            </button>
          </div>

          <div className="max-h-[360px] overflow-y-auto">
            {loading ? (
              <div className="flex items-center justify-center gap-2 px-4 py-10 text-xs text-muted">
                <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
                Memuat notifikasi...
              </div>
            ) : error ? (
              <div className="px-4 py-6 text-center text-xs text-danger">{error}</div>
            ) : items.length === 0 ? (
              <div className="px-4 py-10 text-center">
                <Bell className="mx-auto mb-2 h-6 w-6 text-muted" aria-hidden="true" />
                <p className="text-xs text-muted">Belum ada notifikasi masuk.</p>
              </div>
            ) : (
              <ul className="divide-y divide-border">
                {items.map((item) => (
                  <li key={item.id}>
                    <button
                      type="button"
                      role="menuitem"
                      onClick={() => openNotification(item)}
                      className={cn(
                        'flex w-full items-start gap-3 px-4 py-3 text-left transition-colors hover:bg-background',
                        !item.is_read && 'bg-brand-50/60',
                      )}
                    >
                      <span
                        className={cn(
                          'mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-full',
                          item.is_read ? 'bg-slate-100 text-slate-600' : 'bg-brand-100 text-brand-800',
                        )}
                      >
                        <NotificationIcon type={item.type} className="h-4 w-4" />
                      </span>
                      <span className="min-w-0 flex-1">
                        <span className="flex items-start gap-2">
                          <span className="line-clamp-2 flex-1 text-xs font-semibold text-ink">{item.title}</span>
                          {!item.is_read && (
                            <span className="mt-1 h-2 w-2 shrink-0 rounded-full bg-accent-500" aria-label="Belum dibaca" />
                          )}
                        </span>
                        {item.body && <span className="mt-0.5 line-clamp-2 block text-[11px] text-muted">{item.body}</span>}
                        <span className="mt-1 block text-[10px] text-muted">{timeAgo(item.created_at)}</span>
                      </span>
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </div>

          <div className="border-t border-border p-2">
            <Link
              to="/notifications"
              onClick={() => setOpen(false)}
              className="flex items-center justify-center rounded-control px-3 py-2 text-xs font-semibold text-accent-700 hover:bg-background"
            >
              Lihat semua notifikasi
            </Link>
          </div>
        </div>
      )}
    </div>
  );
}
