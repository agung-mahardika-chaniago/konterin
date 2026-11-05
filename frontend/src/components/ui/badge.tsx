import type { ReactNode } from 'react';

import { cn } from '@/lib/utils';

type BadgeVariant = 'neutral' | 'brand' | 'success' | 'warning' | 'danger' | 'info' | 'violet';

const styles: Record<BadgeVariant, string> = {
  neutral: 'bg-slate-100 text-slate-700 ring-slate-200',
  brand: 'bg-brand-50 text-brand-900 ring-brand-100',
  success: 'bg-emerald-50 text-emerald-700 ring-emerald-200',
  warning: 'bg-amber-50 text-amber-700 ring-amber-200',
  danger: 'bg-red-50 text-red-700 ring-red-200',
  info: 'bg-blue-50 text-blue-700 ring-blue-200',
  violet: 'bg-violet-50 text-violet-700 ring-violet-200',
};

export function Badge({
  variant = 'neutral',
  className,
  children,
}: {
  variant?: BadgeVariant;
  className?: string;
  children: ReactNode;
}) {
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-semibold leading-5 ring-1 ring-inset',
        styles[variant],
        className,
      )}
    >
      {children}
    </span>
  );
}
