import {
  AlertCircle,
  Bell,
  CheckCircle2,
  Clock,
  FlaskConical,
  Inbox,
  Package,
  Search,
  Wrench,
  XCircle,
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';

import { cn } from '@/lib/utils';

export interface StatusConfig {
  label: string;
  icon: LucideIcon;
  classes: string;
}

export const STATUS_CONFIG: Record<string, StatusConfig> = {
  received: {
    label: 'Diterima',
    icon: Inbox,
    classes: 'bg-slate-100 text-slate-800 ring-slate-200',
  },
  waiting_diagnosis: {
    label: 'Menunggu Diagnosis',
    icon: Search,
    classes: 'bg-amber-50 text-amber-800 ring-amber-200',
  },
  waiting_approval: {
    label: 'Menunggu Persetujuan',
    icon: Clock,
    classes: 'bg-orange-50 text-orange-800 ring-orange-200',
  },
  waiting_part: {
    label: 'Menunggu Sparepart',
    icon: Package,
    classes: 'bg-violet-50 text-violet-800 ring-violet-200',
  },
  in_progress: {
    label: 'Sedang Diperbaiki',
    icon: Wrench,
    classes: 'bg-blue-50 text-blue-800 ring-blue-200',
  },
  testing: {
    label: 'Testing',
    icon: FlaskConical,
    classes: 'bg-cyan-50 text-cyan-800 ring-cyan-200',
  },
  completed: {
    label: 'Selesai',
    icon: CheckCircle2,
    classes: 'bg-emerald-50 text-emerald-800 ring-emerald-200',
  },
  ready_for_pickup: {
    label: 'Siap Diambil',
    icon: Bell,
    classes: 'bg-teal-50 text-teal-800 ring-teal-200',
  },
  picked_up: {
    label: 'Sudah Diambil',
    icon: CheckCircle2,
    classes: 'bg-emerald-700 text-white ring-emerald-800',
  },
  cancelled: {
    label: 'Batal',
    icon: XCircle,
    classes: 'bg-red-50 text-red-800 ring-red-200',
  },
};

export function StatusBadge({
  status,
  className,
  showIcon = true,
}: {
  status: string;
  className?: string;
  showIcon?: boolean;
}) {
  const config = STATUS_CONFIG[status] ?? {
    label: status,
    icon: AlertCircle,
    classes: 'bg-slate-100 text-slate-700 ring-slate-200',
  };
  const Icon = config.icon;

  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium leading-5 ring-1 ring-inset',
        config.classes,
        className,
      )}
    >
      {showIcon && <Icon className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />}
      <span>{config.label}</span>
    </span>
  );
}
