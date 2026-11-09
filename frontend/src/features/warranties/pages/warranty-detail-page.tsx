import * as React from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import {
  ArrowLeft,
  CalendarDays,
  MessageCircle,
  ReceiptText,
  ShieldCheck,
  ShieldPlus,
  Smartphone,
  User,
  Wrench,
} from 'lucide-react';

import { EmptyState } from '@/components/shared/empty-state';
import { Alert } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { formatCurrency } from '@/features/dashboard/format';
import { hasPermission } from '@/lib/permissions';
import { useAuthStore } from '@/stores/auth-store';
import { getWarranty, whatsappCustomerLink } from '../api';
import { ClaimCreateDialog } from '../components/claim-create-dialog';
import { ClaimStatusBadge, WarrantyStatusBadge, WarrantyTypeBadge } from '../components/status-badges';
import { daysRemaining, formatDate, formatDateTime } from '../format';
import type { Claim, Warranty } from '../types';

export function WarrantyDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const user = useAuthStore((state) => state.user);
  const canClaim = hasPermission(user, 'warranty.claim');

  const [warranty, setWarranty] = React.useState<Warranty | null>(null);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);
  const [success, setSuccess] = React.useState<string | null>(null);
  const [claimOpen, setClaimOpen] = React.useState(false);

  const loadWarranty = React.useCallback(async () => {
    if (!id) return;
    setLoading(true);
    setError(null);
    try {
      const data = await getWarranty(id);
      setWarranty(data);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Gagal memuat garansi.');
    } finally {
      setLoading(false);
    }
  }, [id]);

  React.useEffect(() => {
    loadWarranty();
  }, [loadWarranty]);

  if (loading) {
    return (
      <div className="space-y-5">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-48 w-full" />
        <div className="grid gap-4 lg:grid-cols-3">
          <Skeleton className="h-64 w-full lg:col-span-2" />
          <Skeleton className="h-64 w-full" />
        </div>
      </div>
    );
  }

  if (error || !warranty) {
    return (
      <div className="space-y-4">
        <Button variant="ghost" onClick={() => navigate('/warranties')} className="gap-2">
          <ArrowLeft className="h-4 w-4" aria-hidden="true" />
          Kembali ke Garansi
        </Button>
        <Alert variant="error">{error || 'Garansi tidak ditemukan.'}</Alert>
      </div>
    );
  }

  const remaining = daysRemaining(warranty.end_date);
  const waUrl = whatsappCustomerLink(
    warranty.customer.phone,
    `Halo ${warranty.customer.name}, kami dari ${warranty.outlet.name} mengenai garansi ${warranty.warranty_no} untuk ${warranty.device.brand} ${warranty.device.model} (tiket ${warranty.service_order.ticket_no}).`,
  );

  const handleClaimCreated = (claim: Claim) => {
    setSuccess(`Klaim ${claim.claim_no} berhasil dibuat.`);
    loadWarranty();
  };

  return (
    <div className="space-y-6 pb-12">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-3">
          <Button
            variant="outline"
            size="sm"
            onClick={() => navigate('/warranties')}
            className="h-9 w-9 rounded-full p-0"
            title="Kembali"
          >
            <ArrowLeft className="h-4 w-4" aria-hidden="true" />
          </Button>
          <div>
            <div className="flex flex-wrap items-center gap-2">
              <h1 className="font-mono text-2xl font-bold tracking-tight text-ink">{warranty.warranty_no}</h1>
              <WarrantyStatusBadge status={warranty.status} />
              <WarrantyTypeBadge type={warranty.type} />
            </div>
            <p className="mt-1 text-xs text-muted">
              Dibuat {formatDateTime(warranty.created_at)} · {warranty.outlet.name}
            </p>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {canClaim && warranty.status === 'active' && (
            <Button size="sm" onClick={() => setClaimOpen(true)} className="gap-1.5">
              <ShieldPlus className="h-4 w-4" aria-hidden="true" />
              Buat Klaim
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

      {success && <Alert variant="success">{success}</Alert>}

      {warranty.status === 'active' && (
        <Card className="border-emerald-200 bg-emerald-50">
          <CardContent className="flex items-start gap-3 p-4">
            <ShieldCheck className="mt-0.5 h-5 w-5 shrink-0 text-emerald-700" aria-hidden="true" />
            <div>
              <p className="text-sm font-semibold text-emerald-900">Garansi aktif</p>
              <p className="text-xs text-emerald-800">
                Berlaku sampai {formatDate(warranty.end_date)}
                {remaining >= 0 ? ` (${remaining} hari lagi)` : ''} · {warranty.days} hari total.
              </p>
            </div>
          </CardContent>
        </Card>
      )}

      <div className="grid gap-6 lg:grid-cols-3">
        <div className="space-y-4 lg:col-span-2">
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-base font-semibold">Ringkasan Garansi</CardTitle>
            </CardHeader>
            <CardContent className="grid gap-4 sm:grid-cols-2">
              <div className="flex items-start gap-3">
                <CalendarDays className="mt-0.5 h-4 w-4 text-accent-600" aria-hidden="true" />
                <div>
                  <p className="text-[11px] font-semibold uppercase tracking-wide text-muted">Masa Berlaku</p>
                  <p className="text-sm font-medium text-ink">
                    {formatDate(warranty.start_date)} – {formatDate(warranty.end_date)}
                  </p>
                  <p className="text-xs text-muted">{warranty.days} hari</p>
                </div>
              </div>
              <div className="flex items-start gap-3">
                <Smartphone className="mt-0.5 h-4 w-4 text-accent-600" aria-hidden="true" />
                <div>
                  <p className="text-[11px] font-semibold uppercase tracking-wide text-muted">Perangkat</p>
                  <p className="text-sm font-medium text-ink">
                    {warranty.device.brand} {warranty.device.model}
                  </p>
                </div>
              </div>
              <div className="flex items-start gap-3">
                <Wrench className="mt-0.5 h-4 w-4 text-accent-600" aria-hidden="true" />
                <div>
                  <p className="text-[11px] font-semibold uppercase tracking-wide text-muted">Servis Asal</p>
                  <Link
                    to={`/services/${warranty.service_order.id}`}
                    className="font-mono text-sm font-semibold text-accent-700 hover:underline"
                  >
                    {warranty.service_order.ticket_no}
                  </Link>
                </div>
              </div>
              {warranty.invoice && (
                <div className="flex items-start gap-3">
                  <ReceiptText className="mt-0.5 h-4 w-4 text-accent-600" aria-hidden="true" />
                  <div>
                    <p className="text-[11px] font-semibold uppercase tracking-wide text-muted">Invoice</p>
                    <Link
                      to={`/invoices/${warranty.invoice.id}`}
                      className="font-mono text-sm font-semibold text-accent-700 hover:underline"
                    >
                      {warranty.invoice.invoice_no}
                    </Link>
                    <p className="text-xs text-muted">{formatCurrency(warranty.invoice.total_amount)}</p>
                  </div>
                </div>
              )}
              {warranty.terms && (
                <div className="sm:col-span-2">
                  <p className="text-[11px] font-semibold uppercase tracking-wide text-muted">Syarat Garansi</p>
                  <p className="mt-1 whitespace-pre-wrap text-sm text-ink">{warranty.terms}</p>
                </div>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between gap-2 pb-2">
              <div>
                <CardTitle className="text-base font-semibold">Riwayat Klaim</CardTitle>
                <p className="text-xs text-muted">Klaim yang diajukan terhadap garansi ini.</p>
              </div>
              {canClaim && warranty.status === 'active' && (
                <Button variant="outline" size="sm" onClick={() => setClaimOpen(true)} className="gap-1.5">
                  <ShieldPlus className="h-3.5 w-3.5" aria-hidden="true" />
                  Klaim Baru
                </Button>
              )}
            </CardHeader>
            <CardContent>
              {!warranty.claims || warranty.claims.length === 0 ? (
                <EmptyState
                  icon={ShieldCheck}
                  title="Belum ada klaim"
                  description="Jika pelanggan melaporkan masalah dalam masa garansi, buat klaim agar penanganan tercatat."
                />
              ) : (
                <div className="divide-y divide-border">
                  {warranty.claims.map((claim) => (
                    <div key={claim.id} className="flex flex-col gap-2 py-3 sm:flex-row sm:items-center sm:justify-between">
                      <div className="min-w-0 space-y-1">
                        <div className="flex flex-wrap items-center gap-2">
                          <Link
                            to={`/claims/${claim.id}`}
                            className="font-mono text-sm font-semibold text-accent-700 hover:underline"
                          >
                            {claim.claim_no}
                          </Link>
                          <ClaimStatusBadge status={claim.status} />
                        </div>
                        <p className="text-sm text-ink">{claim.complaint}</p>
                        <p className="text-xs text-muted">
                          {formatDateTime(claim.claimed_at)}
                          {claim.handled_by ? ` · Ditangani ${claim.handled_by}` : ''}
                        </p>
                        {claim.rework_service_order && (
                          <p className="text-xs text-muted">
                            Rework:{' '}
                            <Link
                              to={`/services/${claim.rework_service_order.id}`}
                              className="font-mono font-semibold text-accent-700 hover:underline"
                            >
                              {claim.rework_service_order.ticket_no}
                            </Link>
                          </p>
                        )}
                      </div>
                      <Link to={`/claims/${claim.id}`}>
                        <Button variant="outline" size="sm">
                          Detail Klaim
                        </Button>
                      </Link>
                    </div>
                  ))}
                </div>
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
                {warranty.customer.name}
              </p>
              <p className="text-xs text-muted">{warranty.customer.phone}</p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-semibold">Status Operasional</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-sm">
              <div className="flex items-center justify-between">
                <span className="text-xs text-muted">Garansi</span>
                <WarrantyStatusBadge status={warranty.status} />
              </div>
              <div className="flex items-center justify-between">
                <span className="text-xs text-muted">Jumlah Klaim</span>
                <span className="font-semibold text-ink">{warranty.claim_count}</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-xs text-muted">Outlet</span>
                <span className="text-right text-xs font-medium text-ink">{warranty.outlet.name}</span>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>

      <ClaimCreateDialog
        open={claimOpen}
        onOpenChange={setClaimOpen}
        warranty={warranty}
        onCreated={handleClaimCreated}
      />
    </div>
  );
}
