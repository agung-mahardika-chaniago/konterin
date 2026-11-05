import { UserRound, Users } from 'lucide-react';

import { EmptyState } from '@/components/shared/empty-state';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';

import { formatDuration, formatNumber, initials } from './format';
import type { DashboardTechnician } from './types';

export function TechnicianPanel({ technicians, days }: { technicians: DashboardTechnician[]; days: number }) {
  return (
    <Card className="overflow-hidden">
      <CardHeader className="pb-3">
        <CardTitle className="text-base">Aktivitas teknisi</CardTitle>
        <CardDescription>Kinerja {days} hari terakhir.</CardDescription>
      </CardHeader>
      <CardContent className="p-0">
        {technicians.length === 0 ? (
          <EmptyState
            icon={Users}
            title="Belum ada aktivitas teknisi"
            description="Aktivitas muncul setelah servis ditugaskan dan dikerjakan teknisi."
          />
        ) : (
          <ul className="max-h-[430px] divide-y divide-border overflow-y-auto">
            {technicians.map((technician) => (
              <li key={technician.id} className="flex items-center gap-3 px-5 py-3">
                <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-brand-50 text-xs font-bold text-brand-900">
                  {initials(technician.name) || <UserRound className="h-4 w-4" aria-hidden="true" />}
                </span>
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-medium text-ink">{technician.name}</p>
                  <p className="text-xs text-muted">
                    {formatNumber(technician.completed_count)} selesai · {formatNumber(technician.active_count)} aktif
                  </p>
                </div>
                <Badge variant={technician.avg_minutes > 0 ? 'brand' : 'neutral'}>
                  ⌀ {formatDuration(technician.avg_minutes)}
                </Badge>
              </li>
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}
