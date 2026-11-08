import * as React from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import {
  ArrowLeft,
  ArrowRightCircle,
  Camera,
  Clock,
  Copy,
  KeyRound,
  Lock,
  MessageCircle,
  Pencil,
  Plus,
  Printer,
  Smartphone,
  Trash2,
  Unlock,
  User,
  UserCog,
  Wallet,
} from 'lucide-react';

import { EmptyState } from '@/components/shared/empty-state';
import { Alert } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { formatCurrency } from '@/features/dashboard/format';
import { InvoiceStatusBadge } from '@/features/invoices/components/invoice-status-badge';
import { PrintLabelModal } from '@/features/print/print-label-modal';
import { hasPermission } from '@/lib/permissions';
import { useAuthStore } from '@/stores/auth-store';
import { deleteServicePhoto, getServiceOrder } from '../api';
import { AssignTechnicianDialog } from '../components/assign-technician-dialog';
import { CredentialFormDialog } from '../components/credential-form-dialog';
import { CredentialRevealDialog } from '../components/credential-reveal-dialog';
import { DiagnosisForm } from '../components/diagnosis-form';
import { ItemsEditor } from '../components/items-editor';
import { PhotoUploadDialog } from '../components/photo-upload-dialog';
import { PriorityBadge } from '../components/priority-badge';
import { ServiceEditDialog } from '../components/service-edit-dialog';
import { StatusBadge } from '../components/status-badge';
import { StatusChangeDialog } from '../components/status-change-dialog';
import { StatusStepper } from '../components/status-stepper';
import { isTerminalStatus } from '../status-flow';
import type { ServiceOrderDetail, ServicePhoto } from '../types';

export function ServiceDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const user = useAuthStore((state) => state.user);

  const [order, setOrder] = React.useState<ServiceOrderDetail | null>(null);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);
  const [copiedLink, setCopiedLink] = React.useState(false);

  // Dialogs
  const [printModalOpen, setPrintModalOpen] = React.useState(false);
  const [editDialogOpen, setEditDialogOpen] = React.useState(false);
  const [credentialFormOpen, setCredentialFormOpen] = React.useState(false);
  const [credentialRevealOpen, setCredentialRevealOpen] = React.useState(false);
  const [photoUploadOpen, setPhotoUploadOpen] = React.useState(false);
  const [statusDialogOpen, setStatusDialogOpen] = React.useState(false);
  const [assignDialogOpen, setAssignDialogOpen] = React.useState(false);

  const loadOrder = React.useCallback(async () => {
    if (!id) return;
    setLoading(true);
    setError(null);
    try {
      const data = await getServiceOrder(id);
      setOrder(data);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Gagal memuat rincian pesanan servis.');
    } finally {
      setLoading(false);
    }
  }, [id]);

  React.useEffect(() => {
    loadOrder();
  }, [loadOrder]);

  const handleDeletePhoto = async (photo: ServicePhoto) => {
    if (!order) return;
    if (!window.confirm('Yakin ingin menghapus foto dokumentasi ini?')) return;
    try {
      await deleteServicePhoto(order.id, photo.id);
      loadOrder();
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : 'Gagal menghapus foto.');
    }
  };

  const handleCopyTrackingLink = () => {
    if (!order?.public_token) return;
    const url = `${window.location.origin}/track/${order.public_token}`;
    navigator.clipboard.writeText(url);
    setCopiedLink(true);
    setTimeout(() => setCopiedLink(false), 2000);
  };

  if (loading) {
    return (
      <div className="space-y-5">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-28 w-full" />
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2 space-y-4">
            <Skeleton className="h-44 w-full" />
            <Skeleton className="h-44 w-full" />
          </div>
          <div className="space-y-4">
            <Skeleton className="h-60 w-full" />
          </div>
        </div>
      </div>
    );
  }

  if (error || !order) {
    return (
      <div className="space-y-4">
        <Button variant="ghost" onClick={() => navigate('/services')} className="gap-2">
          <ArrowLeft className="h-4 w-4" aria-hidden="true" />
          Kembali ke Daftar Servis
        </Button>
        <Alert variant="error">{error || 'Pesanan servis tidak ditemukan.'}</Alert>
      </div>
    );
  }

  const cleanPhone = order.customer.phone.replace(/\D/g, '');
  const waPhone = cleanPhone.startsWith('0') ? '62' + cleanPhone.slice(1) : cleanPhone;
  const waMessage = encodeURIComponent(
    `Halo ${order.customer.name}, kami dari ${order.store.name} menginformasikan mengenai servis HP Anda (${order.device.brand} ${order.device.model}) dengan nomor tiket ${order.ticket_no}.`,
  );
  const waUrl = `https://wa.me/${waPhone}?text=${waMessage}`;

  const canChangeStatus = hasPermission(user, 'service.status') && !isTerminalStatus(order.status);
  const canAssign = hasPermission(user, 'service.assign');
  const canManageDiagnosis = hasPermission(user, 'diagnosis.manage');
  const canViewDiagnosis = canManageDiagnosis || hasPermission(user, 'diagnosis.view');
  const canUpdateItems = hasPermission(user, 'service.update');

  return (
    <div className="space-y-6 pb-12">
      {/* Top Header & Quick Actions */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-3">
          <Button
            variant="outline"
            size="sm"
            onClick={() => navigate('/services')}
            className="h-9 w-9 p-0 rounded-full"
            title="Kembali"
          >
            <ArrowLeft className="h-4 w-4" aria-hidden="true" />
          </Button>
          <div>
            <div className="flex items-center gap-2">
              <h1 className="text-2xl font-bold font-mono tracking-tight text-ink sm:text-[28px]">
                {order.ticket_no}
              </h1>
              <PriorityBadge priority={order.priority} />
            </div>
            <p className="text-xs text-muted mt-0.5">
              Diterima pada{' '}
              {new Date(order.received_at).toLocaleDateString('id-ID', {
                weekday: 'long',
                day: 'numeric',
                month: 'long',
                year: 'numeric',
                hour: '2-digit',
                minute: '2-digit',
              })}
            </p>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {canChangeStatus && (
            <Button size="sm" onClick={() => setStatusDialogOpen(true)} className="gap-1.5 shadow-sm">
              <ArrowRightCircle className="h-4 w-4" aria-hidden="true" />
              Ubah Status
            </Button>
          )}
          <a href={waUrl} target="_blank" rel="noopener noreferrer">
            <Button variant="outline" size="sm" className="gap-1.5 text-emerald-700 hover:text-emerald-800">
              <MessageCircle className="h-4 w-4" aria-hidden="true" />
              Chat Pelanggan
            </Button>
          </a>
          <Button variant="outline" size="sm" onClick={() => setPrintModalOpen(true)} className="gap-1.5">
            <Printer className="h-4 w-4" aria-hidden="true" />
            Cetak Label
          </Button>
          <Button variant="outline" size="sm" onClick={() => setEditDialogOpen(true)} className="gap-1.5">
            <Pencil className="h-4 w-4" aria-hidden="true" />
            Ubah Data
          </Button>
        </div>
      </div>

      {/* Status Stepper */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between pb-2">
          <div>
            <CardTitle className="text-sm font-semibold">Progres Pengerjaan</CardTitle>
            <p className="text-xs text-muted">Alur status servis dari penerimaan hingga serah terima.</p>
          </div>
          <StatusBadge status={order.status} />
        </CardHeader>
        <CardContent>
          <StatusStepper status={order.status} />
        </CardContent>
      </Card>

      {/* Main Grid Layout */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Left 2 Columns: Main Details */}
        <div className="lg:col-span-2 space-y-6">
          {/* Card Pelanggan & Perangkat */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            {/* Pelanggan */}
            <Card>
              <CardHeader className="flex flex-row items-center justify-between pb-2">
                <div className="flex items-center gap-2">
                  <User className="h-4 w-4 text-accent-600" aria-hidden="true" />
                  <CardTitle className="text-sm font-semibold">Data Pelanggan</CardTitle>
                </div>
                {order.customer.id && (
                  <Link to={`/customers/${order.customer.id}`}>
                    <span className="text-[11px] font-semibold text-accent-600 hover:underline">
                      Profil &rarr;
                    </span>
                  </Link>
                )}
              </CardHeader>
              <CardContent className="space-y-1.5 text-sm">
                <p className="font-bold text-ink">{order.customer.name}</p>
                <p className="text-xs text-muted font-mono">{order.customer.phone}</p>
              </CardContent>
            </Card>

            {/* Perangkat */}
            <Card>
              <CardHeader className="flex flex-row items-center gap-2 pb-2">
                <Smartphone className="h-4 w-4 text-accent-600" aria-hidden="true" />
                <CardTitle className="text-sm font-semibold">Data Perangkat</CardTitle>
              </CardHeader>
              <CardContent className="space-y-1 text-sm">
                <p className="font-bold text-ink">
                  {order.device.brand} {order.device.model}
                </p>
                {order.device.color && <p className="text-xs text-muted">Warna: {order.device.color}</p>}
                {order.device.imei && <p className="text-xs text-muted font-mono">IMEI: {order.device.imei}</p>}
              </CardContent>
            </Card>
          </div>

          {/* Keluhan & Kondisi Fisik */}
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-base font-semibold">Keluhan & Catatan Intake</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4 text-sm">
              <div>
                <span className="text-xs font-semibold text-muted uppercase tracking-wider block mb-1">
                  Keluhan / Kerusakan Utama
                </span>
                <p className="text-ink bg-background/60 p-3 rounded-lg border border-border">
                  {order.complaint}
                </p>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <span className="text-xs font-semibold text-muted uppercase tracking-wider block mb-1">
                    Kelengkapan Disertakan
                  </span>
                  <p className="text-xs text-ink">
                    {order.completeness || <span className="italic text-muted">Hanya unit HP</span>}
                  </p>
                </div>
                <div>
                  <span className="text-xs font-semibold text-muted uppercase tracking-wider block mb-1">
                    Kondisi Fisik Unit
                  </span>
                  <p className="text-xs text-ink">
                    {order.physical_condition || <span className="italic text-muted">Kondisi fisik standar</span>}
                  </p>
                </div>
              </div>

              {order.notes && (
                <div>
                  <span className="text-xs font-semibold text-muted uppercase tracking-wider block mb-1">
                    Catatan Pelanggan
                  </span>
                  <p className="text-xs text-ink italic">{order.notes}</p>
                </div>
              )}

              {order.internal_notes && (
                <div className="p-3 rounded-lg bg-amber-50/50 border border-amber-200">
                  <span className="text-xs font-semibold text-amber-900 uppercase tracking-wider block mb-1">
                    🔒 Catatan Internal Teknisi
                  </span>
                  <p className="text-xs text-amber-950">{order.internal_notes}</p>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Diagnosis & Repair Items */}
          {canViewDiagnosis && (
            <DiagnosisForm
              orderId={order.id}
              diagnostic={order.diagnostic ?? null}
              canManage={canManageDiagnosis}
              onSaved={(diagnostic) => setOrder((current) => (current ? { ...current, diagnostic } : current))}
            />
          )}

          <ItemsEditor
            orderId={order.id}
            items={order.items ?? []}
            invoice={order.invoice}
            canUpdate={canUpdateItems}
            onChanged={loadOrder}
          />

          {/* Kredensial Perangkat (Write-Only & Audited Reveal) */}
          <Card>
            <CardHeader className="flex flex-row items-center justify-between pb-3">
              <div className="flex items-center gap-2">
                <KeyRound className="h-5 w-5 text-accent-600" aria-hidden="true" />
                <div>
                  <CardTitle className="text-base font-semibold">Kredensial Layar Kunci</CardTitle>
                  <p className="text-xs text-muted">
                    Terenkripsi AES-256. Dibuka dengan pencatatan audit log keamanan.
                  </p>
                </div>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setCredentialFormOpen(true)}
                className="text-xs"
              >
                {order.credential_configured ? 'Ubah Kredensial' : 'Atur Kredensial'}
              </Button>
            </CardHeader>
            <CardContent>
              {order.credential_configured ? (
                <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between p-4 rounded-lg border border-border bg-background gap-3">
                  <div className="flex items-center gap-3">
                    <div className="h-10 w-10 rounded-full bg-emerald-100 text-emerald-700 flex items-center justify-center shrink-0">
                      <Lock className="h-5 w-5" aria-hidden="true" />
                    </div>
                    <div>
                      <p className="font-semibold text-sm text-ink">
                        Kredensial Tersimpan ({order.credential_type.toUpperCase()})
                      </p>
                      <p className="text-xs text-muted">Nilai kredensial tersembunyi demi privasi pelanggan.</p>
                    </div>
                  </div>
                  <Button
                    size="sm"
                    onClick={() => setCredentialRevealOpen(true)}
                    className="gap-1.5 self-start sm:self-auto"
                  >
                    <Unlock className="h-4 w-4" aria-hidden="true" />
                    Buka Kredensial
                  </Button>
                </div>
              ) : (
                <div className="p-4 rounded-lg border border-dashed border-border text-center text-xs text-muted">
                  Perangkat tidak memiliki kunci layar atau belum dikonfigurasikan.
                </div>
              )}
            </CardContent>
          </Card>

          {/* Foto Dokumentasi */}
          <Card>
            <CardHeader className="flex flex-row items-center justify-between pb-3">
              <div className="flex items-center gap-2">
                <Camera className="h-5 w-5 text-accent-600" aria-hidden="true" />
                <div>
                  <CardTitle className="text-base font-semibold">Foto Dokumentasi</CardTitle>
                  <p className="text-xs text-muted">Foto fisik saat intake, proses pengerjaan, dan pengujian.</p>
                </div>
              </div>
              <Button size="sm" onClick={() => setPhotoUploadOpen(true)} className="gap-1.5 text-xs">
                <Plus className="h-3.5 w-3.5" aria-hidden="true" />
                Unggah Foto
              </Button>
            </CardHeader>
            <CardContent>
              {order.photos.length === 0 ? (
                <EmptyState
                  icon={Camera}
                  title="Belum ada foto dokumentasi"
                  description="Tambahkan foto fisik unit saat diterima untuk mencegah komplain kondisi fisik."
                  className="py-6"
                />
              ) : (
                <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-4">
                  {order.photos.map((p) => (
                    <div key={p.id} className="group relative rounded-lg border border-border overflow-hidden bg-background">
                      <img
                        src={p.file_path}
                        alt={`Dokumentasi ${p.stage}`}
                        className="w-full h-32 object-cover transition-transform group-hover:scale-105"
                      />
                      <div className="p-2 flex items-center justify-between text-xs">
                        <span className="font-semibold uppercase text-[10px] text-muted">{p.stage}</span>
                        <button
                          type="button"
                          onClick={() => handleDeletePhoto(p)}
                          className="text-muted hover:text-danger p-1 rounded transition-colors"
                          title="Hapus foto"
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>

          {/* Status Timeline */}
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-base font-semibold">Riwayat Alur Status</CardTitle>
              <p className="text-xs text-muted">Jejak perpindahan status servis dari penerimaan hingga selesai.</p>
            </CardHeader>
            <CardContent>
              {order.timeline.length === 0 ? (
                <p className="text-xs text-muted">Belum ada catatan riwayat status.</p>
              ) : (
                <div className="relative pl-6 space-y-6 before:absolute before:left-2 before:top-2 before:bottom-2 before:w-0.5 before:bg-border">
                  {order.timeline.map((entry, idx) => (
                    <div key={entry.id || idx} className="relative">
                      <div className="absolute -left-6 top-1 h-2.5 w-2.5 rounded-full bg-accent-500 ring-4 ring-surface" />
                      <div className="space-y-1">
                        <div className="flex items-center gap-2">
                          <StatusBadge status={entry.to_status} />
                          <span className="text-xs text-muted">
                            {new Date(entry.created_at).toLocaleDateString('id-ID', {
                              day: 'numeric',
                              month: 'short',
                              year: 'numeric',
                              hour: '2-digit',
                              minute: '2-digit',
                            })}
                          </span>
                        </div>
                        {entry.changed_by && (
                          <p className="text-xs text-ink font-medium">Oleh: {entry.changed_by}</p>
                        )}
                        {entry.notes && (
                          <p className="text-xs text-muted italic bg-background p-2 rounded border border-border">
                            {entry.notes}
                          </p>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        {/* Right 1 Column: Summary & Actions Sidebar */}
        <div className="space-y-6">
          {/* Card Status & Teknisi */}
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-semibold uppercase tracking-wider text-muted">
                Status Operasional
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-1">
                <span className="text-xs text-muted">Status Saat Ini</span>
                <div>
                  <StatusBadge status={order.status} className="text-sm px-3 py-1" />
                </div>
              </div>

              <div className="space-y-1">
                <span className="text-xs text-muted">Teknisi Bertugas</span>
                <p className="font-semibold text-sm text-ink">
                  {order.assigned_technician ? (
                    order.assigned_technician.name
                  ) : (
                    <span className="text-muted italic">Belum ditugaskan</span>
                  )}
                </p>
              </div>

              <div className="space-y-1">
                <span className="text-xs text-muted">Estimasi Biaya Pengerjaan</span>
                <p className="text-xl font-bold font-mono text-ink">
                  {formatCurrency(order.estimated_cost)}
                </p>
              </div>

              {order.estimated_done_at && (
                <div className="space-y-1">
                  <span className="text-xs text-muted">Target Estimasi Selesai</span>
                  <p className="text-xs font-semibold text-ink flex items-center gap-1.5">
                    <Clock className="h-3.5 w-3.5 text-muted" aria-hidden="true" />
                    {new Date(order.estimated_done_at).toLocaleDateString('id-ID', {
                      weekday: 'short',
                      day: 'numeric',
                      month: 'short',
                      year: 'numeric',
                      hour: '2-digit',
                      minute: '2-digit',
                    })}
                  </p>
                </div>
              )}

              {(canChangeStatus || canAssign) && (
                <div className="space-y-2 border-t border-border pt-3">
                  {canChangeStatus && (
                    <Button size="sm" onClick={() => setStatusDialogOpen(true)} className="w-full gap-1.5">
                      <ArrowRightCircle className="h-3.5 w-3.5" aria-hidden="true" />
                      Ubah Status Servis
                    </Button>
                  )}
                  {canAssign && (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setAssignDialogOpen(true)}
                      className="w-full gap-1.5"
                    >
                      <UserCog className="h-3.5 w-3.5" aria-hidden="true" />
                      Tugaskan Teknisi
                    </Button>
                  )}
                </div>
              )}
            </CardContent>
          </Card>

          {/* Draft Invoice Card */}
          {order.invoice && (
            <Card>
              <CardHeader className="flex flex-row items-center gap-2 pb-2">
                <Wallet className="h-4 w-4 text-accent-600" aria-hidden="true" />
                <CardTitle className="text-sm font-semibold">Tagihan / Invoice</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3 text-sm">
                <div className="flex justify-between items-center text-xs">
                  <span className="text-muted">Nomor Invoice</span>
                  <Link
                    to={`/invoices/${order.invoice.id}`}
                    className="font-mono font-bold text-accent-600 hover:underline"
                  >
                    {order.invoice.invoice_no}
                  </Link>
                </div>
                <div className="flex justify-between items-center text-xs">
                  <span className="text-muted">Status Invoice</span>
                  <InvoiceStatusBadge status={order.invoice.status} />
                </div>
                <div className="flex justify-between items-center text-xs">
                  <span className="text-muted">Sisa Tagihan</span>
                  <span className="font-mono font-semibold text-amber-700">
                    {formatCurrency(order.invoice.remaining_amount)}
                  </span>
                </div>
                <div className="border-t border-border pt-2 flex justify-between items-center font-bold">
                  <span>Total Tagihan</span>
                  <span className="font-mono">{formatCurrency(order.invoice.total_amount)}</span>
                </div>
                <Link to={`/invoices/${order.invoice.id}`}>
                  <Button variant="outline" size="sm" className="w-full gap-1.5 text-xs">
                    <Wallet className="h-3.5 w-3.5" aria-hidden="true" />
                    Buka Invoice & Pembayaran
                  </Button>
                </Link>
              </CardContent>
            </Card>
          )}

          {/* Public Tracking Link */}
          {order.public_token && (
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-semibold">Tracking Publik Pelanggan</CardTitle>
                <p className="text-xs text-muted">Link agar pelanggan dapat melacak progres servis mandiri.</p>
              </CardHeader>
              <CardContent className="space-y-3">
                <div className="p-2.5 rounded bg-background border border-border text-xs font-mono break-all select-all text-slate-700">
                  {order.tracking_url || `/track/${order.public_token}`}
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleCopyTrackingLink}
                  className="w-full gap-2 text-xs"
                >
                  <Copy className="h-3.5 w-3.5" aria-hidden="true" />
                  {copiedLink ? 'Link Tersalin!' : 'Salin Link Tracking'}
                </Button>
              </CardContent>
            </Card>
          )}
        </div>
      </div>

      {/* Dialog Modals */}
      <PrintLabelModal
        open={printModalOpen}
        onOpenChange={setPrintModalOpen}
        order={order}
      />

      <ServiceEditDialog
        open={editDialogOpen}
        onOpenChange={setEditDialogOpen}
        order={order}
        onSuccess={(updated) => setOrder(updated)}
      />

      <CredentialFormDialog
        open={credentialFormOpen}
        onOpenChange={setCredentialFormOpen}
        orderId={order.id}
        hasCredential={order.credential_configured}
        onSuccess={() => loadOrder()}
      />

      <CredentialRevealDialog
        open={credentialRevealOpen}
        onOpenChange={setCredentialRevealOpen}
        orderId={order.id}
        ticketNo={order.ticket_no}
      />

      <PhotoUploadDialog
        open={photoUploadOpen}
        onOpenChange={setPhotoUploadOpen}
        orderId={order.id}
        onSuccess={() => loadOrder()}
      />

      <StatusChangeDialog
        open={statusDialogOpen}
        onOpenChange={setStatusDialogOpen}
        order={order}
        onSuccess={(updated) => setOrder(updated)}
      />

      <AssignTechnicianDialog
        open={assignDialogOpen}
        onOpenChange={setAssignDialogOpen}
        order={order}
        onSuccess={(updated) => setOrder(updated)}
      />
    </div>
  );
}
