import { Loader2 } from 'lucide-react';

export function FullPageLoader({ label = 'Memuat…' }: { label?: string }) {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-3 bg-background text-muted">
      <Loader2 className="h-6 w-6 animate-spin text-brand-900" aria-hidden="true" />
      <p className="text-sm">{label}</p>
    </div>
  );
}
