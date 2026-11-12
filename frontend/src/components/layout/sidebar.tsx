import { PanelLeftClose, PanelLeftOpen, X } from 'lucide-react';
import { NavLink } from 'react-router-dom';

import { BrandMark } from '@/components/shared/brand-mark';
import { navGroups } from '@/components/layout/nav-items';
import { OutletSwitcher } from '@/components/layout/outlet-switcher';
import { hasPermission } from '@/lib/permissions';
import { cn } from '@/lib/utils';
import { useAuthStore } from '@/stores/auth-store';

interface SidebarProps {
  collapsed: boolean;
  mobile?: boolean;
  onToggleCollapse?: () => void;
  onClose?: () => void;
}

export function Sidebar({ collapsed, mobile = false, onToggleCollapse, onClose }: SidebarProps) {
  const user = useAuthStore((state) => state.user);
  const compact = collapsed && !mobile;

  return (
    <aside
      className={cn(
        'flex h-full flex-col bg-brand-950 text-brand-100',
        mobile ? 'w-[280px]' : 'fixed inset-y-0 left-0 z-40 hidden border-r border-white/10 lg:flex',
        compact ? 'lg:w-[76px]' : 'lg:w-[264px]',
      )}
      aria-label="Sidebar aplikasi"
    >
      <div className={cn('flex h-16 shrink-0 items-center border-b border-white/10', compact ? 'justify-center px-3' : 'justify-between px-5')}>
        {compact ? (
          <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-white/10 text-accent-500 ring-1 ring-white/15">
            <span className="text-lg font-bold">F</span>
          </span>
        ) : (
          <BrandMark inverted />
        )}
        {mobile && (
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg p-2 text-brand-100/80 hover:bg-white/10 hover:text-white"
            aria-label="Tutup menu navigasi"
          >
            <X className="h-5 w-5" aria-hidden="true" />
          </button>
        )}
      </div>

      <nav className="flex-1 overflow-y-auto px-3 py-4" aria-label="Navigasi utama">
        <div className="space-y-5">
          {navGroups.map((group) => {
            const items = group.items.filter((item) => !item.permission || hasPermission(user, item.permission));
            if (items.length === 0) return null;

            return (
              <div key={group.label}>
                <p
                  className={cn(
                    'mb-2 px-2 text-[10px] font-semibold uppercase tracking-[0.16em] text-brand-100/45',
                    compact && 'sr-only',
                  )}
                >
                  {group.label}
                </p>
                <ul className="space-y-1">
                  {items.map((item) => {
                    const Icon = item.icon;
                    if (item.to) {
                      return (
                        <li key={item.label}>
                          <NavLink
                            to={item.to}
                            onClick={onClose}
                            title={compact ? item.label : undefined}
                            className={({ isActive }) =>
                              cn(
                                'group relative flex min-h-[44px] items-center gap-3 rounded-control px-2.5 text-sm font-medium transition-colors',
                                compact && 'justify-center px-0',
                                isActive
                                  ? 'bg-brand-700 text-white'
                                  : 'text-brand-100/75 hover:bg-white/5 hover:text-white',
                              )
                            }
                          >
                            {({ isActive }) => (
                              <>
                                {isActive && (
                                  <span
                                    className="absolute left-0 top-1/2 h-5 w-[3px] -translate-y-1/2 rounded-r-full bg-accent-500"
                                    aria-hidden="true"
                                  />
                                )}
                                <Icon className="h-[18px] w-[18px] shrink-0" aria-hidden="true" />
                                <span className={cn('truncate', compact && 'sr-only')}>{item.label}</span>
                              </>
                            )}
                          </NavLink>
                        </li>
                      );
                    }
                    return (
                      <li key={item.label}>
                        <span
                          aria-disabled="true"
                          title={compact ? `${item.label} — Tahap ${item.stage}` : undefined}
                          className={cn(
                            'flex min-h-[44px] cursor-not-allowed items-center gap-3 rounded-control px-2.5 text-sm text-brand-100/40',
                            compact && 'justify-center px-0',
                          )}
                        >
                          <Icon className="h-[18px] w-[18px] shrink-0" aria-hidden="true" />
                          <span className={cn('flex-1 truncate', compact && 'sr-only')}>{item.label}</span>
                          {!compact && (
                            <span className="rounded-full bg-white/5 px-1.5 py-0.5 text-[9px] font-semibold uppercase tracking-wide">
                              Tahap {item.stage}
                            </span>
                          )}
                        </span>
                      </li>
                    );
                  })}
                </ul>
              </div>
            );
          })}
        </div>
      </nav>

      {mobile && (
        <div className="shrink-0 border-t border-white/10 p-3">
          <p className="mb-2 px-1 text-[10px] font-semibold uppercase tracking-[0.16em] text-brand-100/45">
            Outlet aktif
          </p>
          <OutletSwitcher fullWidth />
        </div>
      )}

      <div className={cn('shrink-0 border-t border-white/10 p-3', compact && 'flex justify-center')}>
        {mobile ? (
          <p className="px-2 text-[11px] text-brand-100/40">FixPhone v0.10.0</p>
        ) : (
          <button
            type="button"
            onClick={onToggleCollapse}
            className={cn(
              'flex min-h-[40px] w-full items-center gap-2 rounded-control px-2.5 text-xs text-brand-100/60 hover:bg-white/5 hover:text-white',
              compact && 'justify-center px-0',
            )}
            aria-label={collapsed ? 'Perluas sidebar' : 'Ciutkan sidebar'}
            aria-expanded={!collapsed}
          >
            {collapsed ? (
              <PanelLeftOpen className="h-4 w-4" aria-hidden="true" />
            ) : (
              <PanelLeftClose className="h-4 w-4" aria-hidden="true" />
            )}
            <span className={cn(compact && 'sr-only')}>Ciutkan sidebar</span>
          </button>
        )}
      </div>
    </aside>
  );
}
