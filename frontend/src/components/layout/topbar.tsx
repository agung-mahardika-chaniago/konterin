import { Menu } from 'lucide-react';

import { OutletSwitcher } from '@/components/layout/outlet-switcher';
import { ProfileMenu } from '@/components/layout/profile-menu';
import { NotificationMenu } from '@/features/notifications/components/notification-menu';

export function Topbar({
  onOpenMenu,
  menuOpen,
  title,
  subtitle,
}: {
  onOpenMenu: () => void;
  menuOpen: boolean;
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

        <NotificationMenu />

        <ProfileMenu />
      </div>
    </header>
  );
}
