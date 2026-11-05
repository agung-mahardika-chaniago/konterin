import { Bell, Menu } from 'lucide-react';

import { OutletSwitcher } from '@/components/layout/outlet-switcher';
import { ProfileMenu } from '@/components/layout/profile-menu';

export function Topbar({
  onOpenMenu,
  menuOpen,
  alertCount,
  title,
  subtitle,
}: {
  onOpenMenu: () => void;
  menuOpen: boolean;
  alertCount: number;
  title: string;
  subtitle: string;
}) {
  return (
    <header className="sticky top-0 z-30 border-b border-border bg-surface/95 backdrop-blur supports-[backdrop-filter]:bg-surface/85">
      <div className="flex h-16 items-center gap-2 px-3 sm:gap-3 sm:px-5">
        <button
          type="button"
          onClick={onOpenMenu}
          className="flex h-10 w-10 shrink-0 items-center justify-center rounded-control text-ink hover:bg-background lg:hidden"
          aria-label="Buka menu navigasi"
          aria-expanded={menuOpen}
        >
          <Menu className="h-5 w-5" aria-hidden="true" />
        </button>

        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-semibold text-ink">{title}</p>
          <p className="hidden truncate text-xs text-muted sm:block">{subtitle}</p>
        </div>

        <div className="hidden sm:block">
          <OutletSwitcher />
        </div>

        <a
          href="#alerts-panel"
          className="relative flex h-10 w-10 shrink-0 items-center justify-center rounded-control text-ink hover:bg-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-500"
          aria-label={alertCount > 0 ? `${alertCount} peringatan perlu tindakan` : 'Tidak ada peringatan'}
        >
          <Bell className="h-5 w-5" aria-hidden="true" />
          {alertCount > 0 && (
            <span className="absolute right-1 top-1 flex min-w-[18px] items-center justify-center rounded-full bg-danger px-1 text-[10px] font-bold leading-[18px] text-white">
              {alertCount > 99 ? '99+' : alertCount}
            </span>
          )}
        </a>

        <ProfileMenu />
      </div>
    </header>
  );
}
