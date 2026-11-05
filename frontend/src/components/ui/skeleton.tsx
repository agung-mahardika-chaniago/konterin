import { cn } from '@/lib/utils';

export function Skeleton({ className }: { className?: string }) {
  return <div aria-hidden="true" className={cn('animate-pulse rounded-md bg-slate-200/80', className)} />;
}

export function StatCardSkeleton() {
  return (
    <div className="rounded-card border border-border bg-surface p-4 shadow-card">
      <Skeleton className="h-9 w-9 rounded-lg" />
      <Skeleton className="mt-4 h-3 w-24" />
      <Skeleton className="mt-2 h-7 w-16" />
    </div>
  );
}

export function PanelSkeleton({ rows = 4 }: { rows?: number }) {
  return (
    <div className="space-y-3 p-5 pt-0">
      {Array.from({ length: rows }).map((_, index) => (
        <div key={index} className="flex items-center gap-3">
          <Skeleton className="h-9 w-9 rounded-full" />
          <div className="flex-1 space-y-2">
            <Skeleton className="h-3 w-1/3" />
            <Skeleton className="h-3 w-2/3" />
          </div>
        </div>
      ))}
    </div>
  );
}
