import { Smartphone } from 'lucide-react';

import { cn } from '@/lib/utils';

export function BrandMark({ className, inverted = false }: { className?: string; inverted?: boolean }) {
  return (
    <div className={cn('flex items-center gap-2.5', className)}>
      <span
        className={cn(
          'flex h-10 w-10 items-center justify-center rounded-xl',
          inverted ? 'bg-white/10 text-accent-500 ring-1 ring-white/15' : 'bg-brand-900 text-accent-500',
        )}
      >
        <Smartphone className="h-5 w-5" strokeWidth={1.75} aria-hidden="true" />
      </span>
      <span className="leading-tight">
        <span className={cn('block text-lg font-bold tracking-tight', inverted ? 'text-white' : 'text-brand-950')}>
          FixPhone
        </span>
        <span className={cn('block text-[11px] font-medium uppercase tracking-[0.14em]', inverted ? 'text-brand-100/70' : 'text-muted')}>
          Service Manager
        </span>
      </span>
    </div>
  );
}
