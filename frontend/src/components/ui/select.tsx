import * as React from 'react';

import { cn } from '@/lib/utils';

export type SelectProps = React.SelectHTMLAttributes<HTMLSelectElement>;

export const Select = React.forwardRef<HTMLSelectElement, SelectProps>(({ className, children, ...props }, ref) => (
  <select
    ref={ref}
    className={cn(
      'flex h-10 w-full rounded-control border border-border bg-surface px-3 py-2 text-sm text-ink',
      'focus-visible:border-accent-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 focus-visible:ring-offset-2',
      'disabled:cursor-not-allowed disabled:bg-background disabled:opacity-70',
      className,
    )}
    {...props}
  >
    {children}
  </select>
));
Select.displayName = 'Select';
