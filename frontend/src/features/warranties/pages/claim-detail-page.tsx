import * as React from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import {
  ArrowLeft,
  CheckCircle2,
  CircleDot,
  Clock3,
  MessageCircle,
  Play,
  RotateCcw,
  ShieldCheck,
  ShieldX,
  Smartphone,
  User,
  Wrench,
} from 'lucide-react';

import { Alert } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { hasPermission } from '@/lib/permissions';
import { ApiError } from '@/lib/api-client';
import { useAuthStore } from '@/stores/auth-store';
import { createRework, getClaim, updateClaim, whatsappCustomerLink } from '../api';
import { ClaimResolveDialog } from '../components/claim-resolve-dialog';
import { ClaimStatusBadge } from '../components/status-badges';
import { claimStatusLabel, formatDateTime } from '../format';
import type { Claim, ClaimStatus, ReworkResult } from '../types';

const PROGRESS_STEPS: Array<{ status: ClaimStatus; label: string }> = [
  { status: 'submitted', label: 'Menunggu Review' },
  { status: 'approved', label: 'Disetujui' },
  { status: 'in_progress', label: 'Dikerjakan Ulang' },
  { status: 'completed', label: 'Selesai' },
];

function stepIndex(status: ClaimStatus): number {
  const index = PROGRESS_STEPS.findIndex((step) => step.status === status);
  return index < 0 ? 0 : index;
}

export function ClaimDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const user = useAuthStore((state) => state.user);
  const canClaim = hasPermission(user, 'warranty.claim');
  const canCreateService = hasPermission(user, 'service.create');

  const [claim, setClaim] = React.useState<Claim | null>(null);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);
  const [success, setSuccess] = React.useState<string | null>(null);
  const [actionLoading, setActionLoading] = React.useState<string | null>(null);
  const [resolveTarget, setResolveTarget] = React.useState<'completed' | 'rejected' | null>(null);
  const [rework, setRework] = React.useState<ReworkResult | null>(null);

  const loadClaim = React.useCallback(async () => {
    if (!id) return;
    setLoading(true);
    setError(null);
    try {
      const data = await getClaim(id);
      setClaim(data);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Gagal memuat klaim garansi.');
    } finally {
      setLoading(false);
    }
  }, [id]);

  React.useEffect(() => {
    loadClaim();
  }, [loadClaim]);

  const transition = async (target: Extract<ClaimStatus, 'approved' | 'in_progress'>) => {
    if (!claim) return;
    setActionLoading(target);
    setError(null);
    setSuccess(null);
    try {
      const updated = await updateClaim(claim.id, { status: target });
      setClaim(updated);
      setSuccess(`Status klaim diubah menjadi ${claimStatusLabel(target)}.`);
    } catch (err: unknown) {
      setError(err instanceof ApiError ? err.message : 'Gagal mengubah status klaim.');
    } finally {
      setActionLoading(null);
    }
  };

  const handleRework = async () => {
    if (!claim) return;
    setActionLoading('rework');
    setError(null);
    setSuccess(null);
    try {
      const result = await createRework(claim.id);
      setRework(result);
      setSuccess(`Order rework ${result.ticket_no} berhasil dibuat.`);
      await loadClaim();
    } catch (err: unknown) {
      setError(err instanceof ApiError ? err.message : 'Gagal membuat order rework.');
    } finally {
      setActionLoading(null);
    }
  };

  if (loading) {
    return (
      <div className="space-y-5">
        <Skeleton className="h-8 w-48" />
        <div className="grid gap-4 lg:grid-cols-3">
          <Skeleton className="h-72 w-full lg:col-span-2" />
          <Skeleton className="h-72 w-full" />
        </div>
      </div>
    );
  }

  if (error && !claim) {
    return (
      <div className="space-y-4">
        <Button variant="ghost" onClick={() => navigate('/warranties?tab=claims')} className="gap-2">
          <ArrowLeft className="h-4 w-4" aria-hidden="true" />
          Kembali ke Klaim
        </Button>
        <Alert variant="error">{error}</Alert>
      </div>
    );
  }

  if (!claim) {
    return (
      <div className="space-y-4">
        <Button variant="ghost" onClick={() => navigate('/warranties?tab=claims')} className="gap-2">
          <ArrowLeft className="h-4 w-4" aria-hidden="true" />
          Kembali ke Klaim
        </Button>
        <Alert variant="error">Klaim garansi tidak ditemukan.</Alert>
      </div>
    );
  }

  const final = claim.status === 'completed' || claim.status === 'rejected';
  const reworkOrder = rework
    ? { id: rework.service_order_id, ticket_no: rework.ticket_no }
    : claim.rework_service_order;
  const canRework =
    canClaim && canCreateService && !final && !claim.rework_service_order && !rework;

  const waUrl = whatsappCustomerLink(
    claim.customer.phone,
    `Halo ${claim.customer.name}, kami dari ${claim.outlet.name} mengenai klaim garansi ${claim.claim_no} untuk ${claim.device.brand} ${claim.device.model} (tiket ${claim.service_order.ticket_no}).`,
  );

  const activeStep = stepIndex(claim.status);

  return (
    <div className="space-y-6 pb-12">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-3">
          <Button
            variant="outline"
            size="sm"
            onClick={() => navigate('/warranties?tab=claims')}
            className="h-9 w-9 rounded-full p-0"
            title="Kembali"
          >
            <ArrowLeft className="h-4 w-4" aria-hidden="true" />
          </Button>
          <div>
            <div className="flex flex-wrap items-center gap-2">
              <h1 className="font-mono text-2xl font-bold tracking-tight text-ink">{claim.claim_no}</h1>
              <ClaimStatusBadge status={claim.status} />
            </div>
            <p className="mt-1 text-xs text-muted">
              Diajukan {formatDateTime(claim.claimed_at)} · {claim.outlet.name}
            </p>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {canClaim && claim.status === 'submitted' && (
            <Button
              size="sm"
              onClick={() => transition('approved')}
              loading={actionLoading === 'approved'}
              className="gap-1.5"
            >
              <CheckCircle2 className="h-4 w-4" aria-hidden="true" />
              Setujui Klaim
            </Button>
          )}
          {canClaim && (claim.status === 'submitted' || claim.status === 'approved') && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => transition('in_progress')}
              loading={actionLoading === 'in_progress'}
              className="gap-1.5"
            >
              <Play className="h-4 w-4" aria-hidden="true" />
              Mulai Proses
            </Button>
          )}
          {canClaim && (claim.status === 'approved' || claim.status === 'in_progress') && (
            <Button size="sm" onClick={() => setResolveTarget('completed')} className="gap-1.5">
              <CheckCircle2 className="h-4 w-4" aria-hidden="true" />
              Selesaikan
            </Button>
          )}
          {canClaim && !final && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => setResolveTarget('rejected')}
              className="gap-1.5 text-danger hover:text-danger"
            >
              <ShieldX className="h-4 w-4" aria-hidden="true" />
              Tolak
            </Button>
          )}
          {canRework && (
            <Button
              variant="outline"
              size="sm"
              onClick={handleRework}
              loading={actionLoading === 'rework'}
              className="gap-1.5"
            >
              <RotateCcw className="h-4 w-4" aria-hidden="true" />
              Buat Rework
            </Button>
          )}
          <a href={waUrl} target="_blank" rel="noopener noreferrer">
            <Button variant="outline" size="sm" className="gap-1.5 text-emerald-700 hover:text-emerald-800">
              <MessageCircle className="h-4 w-4" aria-hidden="true" />
              Chat Pelanggan
            </Button>
          </a>
        </div>
      </div>

      {error && <Alert variant="error">{error}</Alert>}
      {success && <Alert variant="success">{success}</Alert>}

      <div className="grid gap-6 lg:grid-cols-3">
        <div className="space-y-4 lg:col-span-2">
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-base font-semibold">Keluhan Pelanggan</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <p className="whitespace-pre-wrap rounded-control border border-border bg-background/60 p-3 text-sm text-ink">
                {claim.complaint}
              </p>

              <div className="grid grid-cols-2 gap-2 sm:grid-cols-4" aria-label="Progres klaim">
                {PROGRESS_STEPS.map((step, index) => {
                  const done = claim.status !== 'rejected' && index <= activeStep;
                  return (
                    <div key={step.status} className="flex flex-col items-center gap-1.5 text-center">
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
                          <CircleDot className="h-4 w-4" aria-hidden="true" />
                        )}
                      </span>
                      <span className={done ? 'text-[10px] font-semibold text-ink' : 'text-[10px] text-muted'}>
                        {step.label}
                      </span>
                    </div>
                  );
                })}
              </div>

              {claim.status === 'rejected' && (
                <Alert variant="error">
                  Klaim ditolak{claim.resolved_at ? ` pada ${formatDateTime(claim.resolved_at)}` : ''}.
                </Alert>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-base font-semibold">Tindak Lanjut</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-start gap-3">
                <Clock3 className="mt-0.5 h-4 w-4 text-accent-600" aria-hidden="true" />
                <div>
                  <p className="text-[11px] font-semibold uppercase tracking-wide text-muted">Status Saat Ini</p>
                  <p className="text-sm font-semibold text-ink">{claimStatusLabel(claim.status)}</p>
                </div>
              </div>

              {claim.resolution ? (
                <div>
                  <p className="text-[11px] font-semibold uppercase tracking-wide text-muted">Resolusi</p>
                  <p className="mt-1 whitespace-pre-wrap rounded-control border border-border bg-background/60 p-3 text-sm text-ink">
                    {claim.resolution}
                  </p>
                </div>
              ) : (
                <p className="text-xs text-muted">
                  Resolusi akan tercatat saat klaim diselesaikan atau ditolak.
                </p>
              )}

              {reworkOrder ? (
                <div className="rounded-control border border-emerald-200 bg-emerald-50 p-3">
                  <p className="flex items-center gap-2 text-sm font-semibold text-emerald-900">
                    <Wrench className="h-4 w-4" aria-hidden="true" />
                    Order rework
                  </p>
                  <p className="mt-1 text-xs text-emerald-800">
                    Perangkat dikerjakan ulang pada tiket{' '}
                    <Link
                      to={`/services/${reworkOrder.id}`}
                      className="font-mono font-semibold text-emerald-900 hover:underline"
                    >
                      {reworkOrder.ticket_no}
                    </Link>
                    .
                  </p>
                </div>
              ) : (
                canRework && (
                  <div className="rounded-control border border-dashed border-border p-3">
                    <p className="text-xs text-muted">
                      Klaim ini belum memiliki order pengerjaan ulang. Buat rework agar teknisi dapat memproses
                      perangkat dengan tiket baru yang tertaut ke klaim.
                    </p>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={handleRework}
                      loading={actionLoading === 'rework'}
                      className="mt-2 gap-1.5"
                    >
                      <RotateCcw className="h-3.5 w-3.5" aria-hidden="true" />
                      Buat Order Rework
                    </Button>
                  </div>
                )
              )}
            </CardContent>
          </Card>
        </div>

        <div className="space-y-4">
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-semibold">Pelanggan</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2 text-sm">
              <p className="flex items-center gap-2 font-medium text-ink">
                <User className="h-4 w-4 text-muted" aria-hidden="true" />
                {claim.customer.name}
              </p>
              <p className="text-xs text-muted">{claim.customer.phone}</p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-semibold">Perangkat & Servis</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-sm">
              <p className="flex items-center gap-2 font-medium text-ink">
                <Smartphone className="h-4 w-4 text-muted" aria-hidden="true" />
                {claim.device.brand} {claim.device.model}
              </p>
              <div>
                <p className="text-[11px] font-semibold uppercase tracking-wide text-muted">Servis Asal</p>
                <Link
                  to={`/services/${claim.service_order.id}`}
                  className="font-mono text-sm font-semibold text-accent-700 hover:underline"
                >
                  {claim.service_order.ticket_no}
                </Link>
              </div>
              <div>
                <p className="text-[11px] font-semibold uppercase tracking-wide text-muted">Garansi</p>
                <Link
                  to={`/warranties/${claim.warranty_id}`}
                  className="inline-flex items-center gap-1 font-mono text-sm font-semibold text-accent-700 hover:underline"
                >
                  <ShieldCheck className="h-3.5 w-3.5" aria-hidden="true" />
                  {claim.warranty_no}
                </Link>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-semibold">Riwayat Penanganan</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-sm">
              <div className="flex items-center justify-between gap-2">
                <span className="text-xs text-muted">Diajukan</span>
                <span className="text-right text-xs font-medium text-ink">{formatDateTime(claim.claimed_at)}</span>
              </div>
              <div className="flex items-center justify-between gap-2">
                <span className="text-xs text-muted">Ditangani oleh</span>
                <span className="text-right text-xs font-medium text-ink">{claim.handled_by ?? '—'}</span>
              </div>
              <div className="flex items-center justify-between gap-2">
                <span className="text-xs text-muted">Diselesaikan</span>
                <span className="text-right text-xs font-medium text-ink">{formatDateTime(claim.resolved_at)}</span>
              </div>
              <div className="flex items-center justify-between gap-2">
                <span className="text-xs text-muted">Outlet</span>
                <span className="text-right text-xs font-medium text-ink">{claim.outlet.name}</span>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>

      <ClaimResolveDialog
        open={resolveTarget !== null}
        onOpenChange={(open) => {
          if (!open) setResolveTarget(null);
        }}
        claim={claim}
        targetStatus={resolveTarget ?? 'completed'}
        onUpdated={(updated) => {
          setClaim(updated);
          setSuccess(`Klaim ${updated.claim_no} telah ${updated.status === 'completed' ? 'diselesaikan' : 'ditolak'}.`);
        }}
      />
    </div>
  );
}
