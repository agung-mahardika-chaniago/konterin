import { Check, XCircle } from 'lucide-react';

import { cn } from '@/lib/utils';
import { BRANCH_STATUSES, MAIN_STATUS_FLOW } from '../status-flow';
import { STATUS_CONFIG } from './status-badge';

interface StatusStepperProps {
  status: string;
}

export function StatusStepper({ status }: StatusStepperProps) {
  if (status === 'cancelled') {
    return (
      <div className="flex items-center gap-2 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800">
        <XCircle className="h-4 w-4" aria-hidden="true" />
        <span className="font-semibold">Servis dibatalkan.</span>
        <span className="text-xs text-red-700">Tidak ada transisi status berikutnya.</span>
      </div>
    );
  }

  const isBranch = (BRANCH_STATUSES as readonly string[]).includes(status);
  const currentIndex = isBranch ? 1 : MAIN_STATUS_FLOW.indexOf(status as (typeof MAIN_STATUS_FLOW)[number]);
  const branchConfig = isBranch ? STATUS_CONFIG[status] : null;

  return (
    <div className="space-y-3">
      <div className="overflow-x-auto pb-1">
        <ol className="flex min-w-[620px] items-start">
          {MAIN_STATUS_FLOW.map((step, index) => {
            const config = STATUS_CONFIG[step];
            const Icon = config?.icon;
            const done = currentIndex >= 0 && index < currentIndex;
            const current = index === currentIndex;

            return (
              <li key={step} className="flex flex-1 flex-col items-center relative">
                {index > 0 && (
                  <span
                    className={cn(
                      'absolute right-1/2 top-4 h-0.5 w-full -translate-y-1/2',
                      done || current ? 'bg-accent-500' : 'bg-border',
                    )}
                    aria-hidden="true"
                  />
                )}
                <span
                  className={cn(
                    'relative z-10 flex h-8 w-8 items-center justify-center rounded-full border-2 text-xs font-semibold transition-colors',
                    done && 'border-accent-500 bg-accent-500 text-white',
                    current && 'border-accent-500 bg-surface text-accent-600 ring-4 ring-accent-100',
                    !done && !current && 'border-border bg-surface text-muted',
                  )}
                >
                  {done ? (
                    <Check className="h-4 w-4" aria-hidden="true" />
                  ) : Icon ? (
                    <Icon className="h-4 w-4" aria-hidden="true" />
                  ) : (
                    index + 1
                  )}
                </span>
                <span
                  className={cn(
                    'mt-2 max-w-[86px] text-center text-[11px] leading-tight',
                    current ? 'font-semibold text-ink' : 'text-muted',
                  )}
                >
                  {config?.label ?? step}
                </span>
              </li>
            );
          })}
        </ol>
      </div>

      {branchConfig && (
        <p className="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-900">
          Status saat ini berada di cabang <span className="font-semibold">{branchConfig.label}</span>. Lanjutkan ke
          pengerjaan setelah persetujuan pelanggan atau sparepart tersedia.
        </p>
      )}
    </div>
  );
}
