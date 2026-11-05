import type { LucideIcon } from 'lucide-react';
import type { ReactNode } from 'react';

import { cn } from '@/lib/utils';

export function EmptyState({
  icon: Icon,
  title,
  description,
  action,
  className,
}: {
  icon?: LucideIcon;
  title: string;
  description?: string;
  action?: ReactNode;
  className?: string;
}) {
  return (
    <div className={cn('flex flex-col items-center justify-center px-4 py-8 text-center', className)}>
      {Icon && (
        <span className="mb-3 flex h-11 w-11 items-center justify-center rounded-full bg-background text-muted">
          <Icon className="h-5 w-5" aria-hidden="true" />
        </span>
      )}
      <p className="text-sm font-medium text-ink">{title}</p>
      {description && <p className="mt-1 max-w-sm text-xs leading-5 text-muted">{description}</p>}
      {action && <div className="mt-4">{action}</div>}
    </div>
  );
}
