import * as React from 'react';
import { AlertCircle, CheckCircle2, Info } from 'lucide-react';

import { cn } from '@/lib/utils';

type AlertVariant = 'info' | 'error' | 'success';

const styles: Record<AlertVariant, string> = {
  info: 'border-brand-100 bg-brand-50 text-brand-900',
  error: 'border-red-200 bg-red-50 text-red-800',
  success: 'border-emerald-200 bg-emerald-50 text-emerald-800',
};

const icons: Record<AlertVariant, React.ComponentType<{ className?: string }>> = {
  info: Info,
  error: AlertCircle,
  success: CheckCircle2,
};

export function Alert({
  variant = 'info',
  className,
  children,
}: {
  variant?: AlertVariant;
  className?: string;
  children: React.ReactNode;
}) {
  const Icon = icons[variant];
  return (
    <div role={variant === 'error' ? 'alert' : 'status'} className={cn('flex gap-2 rounded-control border p-3 text-sm', styles[variant], className)}>
      <Icon className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
      <div className="min-w-0">{children}</div>
    </div>
  );
}
