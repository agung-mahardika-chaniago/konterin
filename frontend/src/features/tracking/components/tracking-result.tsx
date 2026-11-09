import { CheckCircle2, Clock, PackageCheck, ShieldCheck, Smartphone, Store } from 'lucide-react';

import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import type { PublicTracking } from '../types';

const PROGRESS_STEPS = [
  { key: 'received', label: 'Diterima' },
  { key: 'waiting_diagnosis', label: 'Diagnosis' },
  { key: 'in_progress', label: 'Dikerjakan' },
  { key: 'testing', label: 'Diuji' },
  { key: 'completed', label: 'Selesai' },
  { key: 'ready_for_pickup', label: 'Siap Diambil' },
  { key: 'picked_up', label: 'Diambil' },
];

const STATUS_ORDER: Record<string, number> = {
  received: 0,
  waiting_diagnosis: 1,
  waiting_approval: 2,
  waiting_part: 2,
  in_progress: 2,
  testing: 3,
  completed: 4,
  ready_for_pickup: 5,
  picked_up: 6,
  cancelled: -1,
};

function formatDate(value?: string | null): string {
  if (!value) return '—';
  return new Date(value).toLocaleDateString('id-ID', {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function statusTone(status: string): 'success' | 'warning' | 'info' | 'neutral' | 'danger' {
  if (status === 'picked_up' || status === 'completed') return 'success';
  if (status === 'ready_for_pickup') return 'info';
  if (status === 'cancelled') return 'danger';
  if (status === 'waiting_approval' || status === 'waiting_part') return 'warning';
  return 'neutral';
}

export function TrackingResult({ data }: { data: PublicTracking }) {
  const currentIndex = STATUS_ORDER[data.status] ?? 0;
  const cancelled = data.status === 'cancelled';

  return (
    <div className="space-y-5">
      <Card>
        <CardHeader className="flex flex-row items-start justify-between gap-3">
          <div>
            <div className="flex items-center gap-2 text-xs font-medium uppercase tracking-wide text-muted">
              <Store className="h-3.5 w-3.5" aria-hidden="true" />
              {data.store.name}
            </div>
            <CardTitle className="mt-1 font-mono text-xl">{data.ticket_no}</CardTitle>
            <p className="mt-1 text-xs text-muted">Diterima {formatDate(data.received_at)}</p>
          </div>
          <Badge variant={statusTone(data.status)} className="text-xs">
            {data.status_label}
          </Badge>
        </CardHeader>
        <CardContent className="space-y-5">
          {!cancelled && (
            <ol className="grid grid-cols-4 gap-2 sm:grid-cols-7" aria-label="Progres servis">
              {PROGRESS_STEPS.map((step, index) => {
                const done = index <= currentIndex;
                return (
                  <li key={step.key} className="flex flex-col items-center gap-1.5 text-center">
                    <span
                      className={
                        done
                          ? 'flex h-7 w-7 items-center justify-center rounded-full bg-emerald-600 text-white'
                          : 'flex h-7 w-7 items-center justify-center rounded-full border border-border bg-background text-muted'
                      }
                    >
                      {done ? (
                        <CheckCircle2 className="h-4 w-4" aria-hidden="true" />
                      ) : (
                        <span className="text-[10px] font-semibold">{index + 1}</span>
                      )}
                    </span>
                    <span className={done ? 'text-[10px] font-semibold text-ink' : 'text-[10px] text-muted'}>
                      {step.label}
                    </span>
                  </li>
                );
              })}
            </ol>
          )}

          <div className="grid gap-3 sm:grid-cols-2">
            <div className="rounded-control border border-border bg-background/60 p-3">
              <p className="text-[11px] font-semibold uppercase tracking-wide text-muted">Perangkat</p>
              <p className="mt-1 flex items-center gap-1.5 text-sm font-semibold text-ink">
                <Smartphone className="h-4 w-4 text-accent-600" aria-hidden="true" />
                {data.device}
              </p>
            </div>
            <div className="rounded-control border border-border bg-background/60 p-3">
              <p className="text-[11px] font-semibold uppercase tracking-wide text-muted">Status Pengambilan</p>
              <p className="mt-1 flex items-center gap-1.5 text-sm font-semibold text-ink">
                <PackageCheck className="h-4 w-4 text-accent-600" aria-hidden="true" />
                {data.pickup_status}
              </p>
            </div>
            <div className="rounded-control border border-border bg-background/60 p-3">
              <p className="text-[11px] font-semibold uppercase tracking-wide text-muted">Estimasi Selesai</p>
              <p className="mt-1 flex items-center gap-1.5 text-sm font-semibold text-ink">
                <Clock className="h-4 w-4 text-accent-600" aria-hidden="true" />
                {formatDate(data.estimated_done_at)}
              </p>
            </div>
            <div className="rounded-control border border-border bg-background/60 p-3">
              <p className="text-[11px] font-semibold uppercase tracking-wide text-muted">Pembayaran</p>
              <p className="mt-1 text-sm font-semibold text-ink">{data.payment_status_label}</p>
            </div>
          </div>

          <div>
            <p className="text-[11px] font-semibold uppercase tracking-wide text-muted">Keluhan</p>
            <p className="mt-1 rounded-control border border-border bg-background/60 p-3 text-sm text-ink">
              {data.complaint}
            </p>
          </div>

          {data.diagnosis && (
            <div>
              <p className="text-[11px] font-semibold uppercase tracking-wide text-muted">Hasil Diagnosis</p>
              <p className="mt-1 rounded-control border border-border bg-background/60 p-3 text-sm text-ink">
                {data.diagnosis}
              </p>
            </div>
          )}

          {data.warranty.active && (
            <div className="flex items-start gap-3 rounded-control border border-emerald-200 bg-emerald-50 p-3">
              <ShieldCheck className="mt-0.5 h-4 w-4 shrink-0 text-emerald-700" aria-hidden="true" />
              <div>
                <p className="text-xs font-semibold text-emerald-800">Garansi aktif</p>
                <p className="text-xs text-emerald-700">
                  Berlaku {data.warranty.days} hari sampai {data.warranty.end_date ?? '—'}.
                </p>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
      <p className="text-center text-[11px] text-muted">
        Halaman ini hanya menampilkan informasi progres servis. Hubungi toko untuk pertanyaan lebih lanjut.
      </p>
    </div>
  );
}
