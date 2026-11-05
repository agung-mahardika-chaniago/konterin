import { ChevronDown, KeyRound, LogOut, UserRound } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';

import { useAuthStore } from '@/stores/auth-store';
import { cn } from '@/lib/utils';

const roleLabels: Record<string, string> = {
  super_admin: 'Super Admin',
  owner: 'Owner',
  admin: 'Admin',
  technician: 'Teknisi',
  cashier: 'Kasir',
};

function initials(name: string): string {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase() ?? '')
    .join('');
}

export function ProfileMenu() {
  const user = useAuthStore((state) => state.user);
  const logout = useAuthStore((state) => state.logout);
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const [loggingOut, setLoggingOut] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;

    const onPointerDown = (event: MouseEvent) => {
      if (!rootRef.current?.contains(event.target as Node)) setOpen(false);
    };
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setOpen(false);
        triggerRef.current?.focus();
      }
    };

    document.addEventListener('mousedown', onPointerDown);
    document.addEventListener('keydown', onKeyDown);
    const first = menuRef.current?.querySelector<HTMLElement>('[role="menuitem"]');
    first?.focus();

    return () => {
      document.removeEventListener('mousedown', onPointerDown);
      document.removeEventListener('keydown', onKeyDown);
    };
  }, [open]);

  if (!user) return null;

  const handleMenuKeyDown = (event: React.KeyboardEvent<HTMLDivElement>) => {
    const items = Array.from(menuRef.current?.querySelectorAll<HTMLElement>('[role="menuitem"]') ?? []);
    const current = items.indexOf(document.activeElement as HTMLElement);
    if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
      event.preventDefault();
      const next = event.key === 'ArrowDown' ? (current + 1) % items.length : (current - 1 + items.length) % items.length;
      items[next]?.focus();
    }
    if (event.key === 'Home') {
      event.preventDefault();
      items[0]?.focus();
    }
    if (event.key === 'End') {
      event.preventDefault();
      items[items.length - 1]?.focus();
    }
  };

  async function handleLogout() {
    setLoggingOut(true);
    setOpen(false);
    await logout();
    navigate('/login', { replace: true });
  }

  return (
    <div ref={rootRef} className="relative">
      <button
        ref={triggerRef}
        type="button"
        onClick={() => setOpen((value) => !value)}
        className="flex min-h-[44px] items-center gap-2 rounded-control px-1.5 hover:bg-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-500"
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls="profile-menu"
      >
        <span className="flex h-9 w-9 items-center justify-center rounded-full bg-brand-900 text-xs font-semibold text-white">
          {initials(user.name) || <UserRound className="h-4 w-4" aria-hidden="true" />}
        </span>
        <span className="hidden text-left sm:block">
          <span className="block max-w-[150px] truncate text-xs font-semibold leading-4 text-ink">{user.name}</span>
          <span className="block text-[11px] leading-4 text-muted">{roleLabels[user.role] ?? user.role}</span>
        </span>
        <ChevronDown className={cn('h-4 w-4 text-muted transition-transform', open && 'rotate-180')} aria-hidden="true" />
      </button>

      {open && (
        <div
          ref={menuRef}
          id="profile-menu"
          role="menu"
          aria-label="Menu profil"
          onKeyDown={handleMenuKeyDown}
          className="absolute right-0 top-[calc(100%+8px)] z-50 w-64 overflow-hidden rounded-card border border-border bg-surface shadow-modal"
        >
          <div className="border-b border-border px-4 py-3">
            <p className="truncate text-sm font-semibold text-ink">{user.name}</p>
            <p className="truncate text-xs text-muted">{user.email}</p>
            <p className="mt-1 text-[11px] font-medium text-brand-600">{roleLabels[user.role] ?? user.role}</p>
          </div>
          <div className="p-1.5">
            <button
              type="button"
              role="menuitem"
              onClick={() => {
                setOpen(false);
                navigate('/change-password');
              }}
              className="flex w-full items-center gap-2 rounded-control px-3 py-2.5 text-left text-sm text-ink hover:bg-background focus-visible:bg-background focus-visible:outline-none"
            >
              <KeyRound className="h-4 w-4 text-muted" aria-hidden="true" />
              Ubah password
            </button>
            <button
              type="button"
              role="menuitem"
              onClick={() => void handleLogout()}
              disabled={loggingOut}
              className="flex w-full items-center gap-2 rounded-control px-3 py-2.5 text-left text-sm text-danger hover:bg-red-50 focus-visible:bg-red-50 focus-visible:outline-none disabled:opacity-60"
            >
              <LogOut className="h-4 w-4" aria-hidden="true" />
              {loggingOut ? 'Keluar…' : 'Keluar'}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
