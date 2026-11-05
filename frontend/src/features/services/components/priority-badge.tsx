import { Zap } from 'lucide-react';

import { cn } from '@/lib/utils';

export function PriorityBadge({
  priority,
  className,
}: {
  priority: string;
  className?: string;
}) {
  const isExpress = priority === 'express';

  if (!isExpress) {
    return (
      <span
        className={cn(
          'inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium leading-4 bg-slate-100 text-slate-600 ring-1 ring-inset ring-slate-200',
          className,
        )}
      >
        Normal
      </span>
    );
  }

  return (
    <span
      className={cn(
        'inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-semibold leading-4 bg-amber-100 text-amber-900 ring-1 ring-inset ring-amber-300',
        className,
      )}
    >
      <Zap className="h-3 w-3 fill-amber-500 text-amber-600" aria-hidden="true" />
      Express
    </span>
  );
}
