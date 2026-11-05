import type { LucideIcon } from 'lucide-react';
import type { ReactNode } from 'react';

import { cn } from '@/lib/utils';

type Tone = 'brand' | 'warning' | 'info' | 'success' | 'danger' | 'accent' | 'violet';

const tones: Record<Tone, string> = {
  brand: 'bg-brand-50 text-brand-700',
  warning: 'bg-amber-50 text-amber-600',
  info: 'bg-blue-50 text-blue-600',
  success: 'bg-emerald-50 text-emerald-600',
  danger: 'bg-red-50 text-red-600',
  accent: 'bg-accent-50 text-accent-600',
  violet: 'bg-violet-50 text-violet-600',
};

export function StatCard({
  icon: Icon,
  label,
  value,
  hint,
  tone = 'brand',
  className,
}: {
  icon: LucideIcon;
  label: string;
  value: ReactNode;
  hint?: string;
  tone?: Tone;
  className?: string;
}) {
  return (
    <div className={cn('rounded-card border border-border bg-surface p-4 shadow-card', className)}>
      <div className={cn('flex h-9 w-9 items-center justify-center rounded-lg', tones[tone])}>
        <Icon className="h-[18px] w-[18px]" aria-hidden="true" />
      </div>
      <p className="mt-3 text-xs font-medium leading-5 text-muted">{label}</p>
      <p className="tabular-nums mt-0.5 text-[22px] font-bold leading-8 tracking-tight text-ink">{value}</p>
      {hint && <p className="mt-0.5 text-[11px] leading-4 text-muted">{hint}</p>}
    </div>
  );
}
