import { ChevronDown, Store } from 'lucide-react';

import { useAuthStore } from '@/stores/auth-store';
import { useOutletStore } from '@/stores/outlet-store';
import { cn } from '@/lib/utils';

export function OutletSwitcher({ fullWidth = false }: { fullWidth?: boolean }) {
  const user = useAuthStore((state) => state.user);
  const outletId = useOutletStore((state) => state.outletId);
  const setOutletId = useOutletStore((state) => state.setOutletId);

  if (!user) return null;
  const outlets = user.outlets;
  const active = outlets.find((outlet) => outlet.id === outletId);

  if (outlets.length === 0) {
    return (
      <span
        className={cn(
          'items-center gap-2 rounded-control border border-border bg-surface px-3 py-2 text-xs text-muted',
          fullWidth ? 'flex w-full' : 'hidden sm:inline-flex',
        )}
      >
        <Store className="h-4 w-4" aria-hidden="true" />
        Tanpa outlet
      </span>
    );
  }

  if (outlets.length === 1) {
    return (
      <span
        className={cn(
          'items-center gap-2 rounded-control border border-border bg-surface px-3 py-2 text-xs font-medium text-ink',
          fullWidth ? 'flex w-full' : 'hidden sm:inline-flex',
        )}
      >
        <Store className="h-4 w-4 shrink-0 text-brand-600" aria-hidden="true" />
        <span className="truncate">{active?.name ?? outlets[0].name}</span>
        <span className="ml-auto rounded bg-background px-1.5 py-0.5 text-[10px] font-semibold text-muted">
          {outlets[0].code}
        </span>
      </span>
    );
  }

  return (
    <div className={cn('relative', fullWidth ? 'w-full' : 'w-[150px] sm:w-[190px]')}>
      <label htmlFor={fullWidth ? 'outlet-switcher-mobile' : 'outlet-switcher'} className="sr-only">
        Pilih outlet aktif
      </label>
      <Store
        className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-brand-600"
        aria-hidden="true"
      />
      <select
        id={fullWidth ? 'outlet-switcher-mobile' : 'outlet-switcher'}
        value={outletId ?? ''}
        onChange={(event) => setOutletId(event.target.value || null)}
        className="h-10 w-full appearance-none rounded-control border border-border bg-surface pl-9 pr-8 text-xs font-medium text-ink shadow-sm hover:border-brand-100 focus-visible:border-accent-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-500/40"
      >
        {outlets.map((outlet) => (
          <option key={outlet.id} value={outlet.id}>
            {outlet.name} ({outlet.code})
          </option>
        ))}
      </select>
      <ChevronDown
        className="pointer-events-none absolute right-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted"
        aria-hidden="true"
      />
    </div>
  );
}
