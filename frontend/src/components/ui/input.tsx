import * as React from 'react';

import { cn } from '@/lib/utils';

export type InputProps = React.InputHTMLAttributes<HTMLInputElement>;

export const Input = React.forwardRef<HTMLInputElement, InputProps>(({ className, type, ...props }, ref) => (
  <input
    ref={ref}
    type={type}
    className={cn(
      'flex h-10 w-full rounded-control border border-border bg-surface px-3 py-2 text-sm text-ink',
      'placeholder:text-muted/70 focus-visible:border-accent-500 focus-visible:outline-none',
      'focus-visible:ring-2 focus-visible:ring-accent-500 focus-visible:ring-offset-2',
      'disabled:cursor-not-allowed disabled:bg-background disabled:opacity-70',
      className,
    )}
    {...props}
  />
));
Input.displayName = 'Input';
