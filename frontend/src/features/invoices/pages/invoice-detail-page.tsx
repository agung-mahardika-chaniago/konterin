import * as React from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import {
  ArrowLeft,
  BadgePercent,
  Ban,
  Banknote,
  CheckCircle2,
  MessageCircle,
  Printer,
  ReceiptText,
  Wallet,
} from 'lucide-react';

import { EmptyState } from '@/components/shared/empty-state';
import { Alert } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { formatCurrency } from '@/features/dashboard/format';
import { PrintInvoiceModal } from '@/features/print/print-invoice-modal';
import type { PrintTemplate } from '@/features/print/print-invoice-modal';
import { ApiError } from '@/lib/api-client';
import { hasPermission } from '@/lib/permissions';
import { useAuthStore } from '@/stores/auth-store';
import { getInvoice, issueInvoice, voidInvoice, voidPayment, whatsappInvoiceLink } from '../api';
import { InvoiceDiscountDialog } from '../components/invoice-discount-dialog';
import { InvoiceStatusBadge } from '../components/invoice-status-badge';
import { PaymentDialog } from '../components/payment-dialog';
import { VoidReasonDialog } from '../components/void-reason-dialog';
import {
  PAYMENT_STATUS,
  formatDate,
  formatDateTime,
  invoiceStatusLabel,
  itemTypeLabel,
  paymentMethodLabel,
} from '../format';
import type { InvoiceDetail, InvoicePayment, PaymentMutation } from '../types';

export function InvoiceDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const user = useAuthStore((state) => state.user);

  const [invoice, setInvoice] = React.useState<InvoiceDetail | null>(null);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);
  const [success, setSuccess] = React.useState<string | null>(null);
  const [issuing, setIssuing] = React.useState(false);

  const [paymentOpen, setPaymentOpen] = React.useState(false);
  const [discountOpen, setDiscountOpen] = React.useState(false);
  const [voidInvoiceOpen, setVoidInvoiceOpen] = React.useState(false);
  const [voidPaymentTarget, setVoidPaymentTarget] = React.useState<InvoicePayment | null>(null);
  const [printOpen, setPrintOpen] = React.useState(false);
  const [printTemplate, setPrintTemplate] = React.useState<PrintTemplate>('a4');

  const loadInvoice = React.useCallback(
    async (showLoader = true) => {
      if (!id) return;
      if (showLoader) setLoading(true);
      setError(null);
      try {
        const data = await getInvoice(id);
        setInvoice(data);
      } catch (err: unknown) {
        setError(err instanceof Error ? err.message : 'Gagal memuat rincian invoice.');
      } finally {
        if (showLoader) setLoading(false);
      }
    },
    [id],
  );

  React.useEffect(() => {
    loadInvoice(true);
  }, [loadInvoice]);

  const openPrint = (template: PrintTemplate) => {
    setPrintTemplate(template);
    setPrintOpen(true);
  };

  const handleIssue = async () => {
    if (!invoice) return;
    const confirmed = window.confirm(
      `Finalisasi invoice ${invoice.invoice_no}?\n\nItem servis akan disalin sebagai snapshot dan diskon dikunci setelah finalisasi.`,
    );
    if (!confirmed) return;
    setIssuing(true);
    setError(null);
    setSuccess(null);
    try {
      await issueInvoice(invoice.id);
      setSuccess('Invoice berhasil difinalisasi dan siap dibayar.');
      await loadInvoice(false);
    } catch (err: unknown) {
      setError(err instanceof ApiError ? err.message : 'Gagal memfinalisasi invoice.');
    } finally {
      setIssuing(false);
    }
  };

  const handlePaymentSuccess = (result: PaymentMutation) => {
    setSuccess(`Pembayaran ${formatCurrency(result.payment.amount)} berhasil dicatat.`);
    loadInvoice(false);
  };

  const handleVoidInvoice = async (reason: string) => {
    if (!invoice) return;
    await voidInvoice(invoice.id, reason);
    setSuccess('Invoice berhasil dibatalkan.');
    await loadInvoice(false);
  };

  const handleVoidPayment = async (reason: string) => {
    if (!voidPaymentTarget) return;
    const result = await voidPayment(voidPaymentTarget.id, reason);
    setSuccess(`Pembayaran ${result.payment.payment_no} berhasil dibatalkan.`);
    await loadInvoice(false);
  };

  if (loading) {
    return (
      <div className="space-y-5">
        <Skeleton className="h-8 w-56" />
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
          <div className="space-y-4 lg:col-span-2">
            <Skeleton className="h-72 w-full" />
            <Skeleton className="h-52 w-full" />
          </div>
          <div className="space-y-4">
            <Skeleton className="h-56 w-full" />
            <Skeleton className="h-40 w-full" />
          </div>
        </div>
      </div>
    );
  }

  if (error || !invoice) {
    return (
      <div className="space-y-4">
        <Button variant="ghost" onClick={() => navigate('/invoices')} className="gap-2">
          <ArrowLeft className="h-4 w-4" aria-hidden="true" />
          Kembali ke Daftar Invoice
        </Button>
        <Alert variant="error">{error || 'Invoice tidak ditemukan.'}</Alert>
      </div>
    );
  }

  const isDraft = invoice.status === 'draft';
  const isVoid = invoice.status === 'void';
  const canIssue = hasPermission(user, 'invoice.create') && isDraft;
  const canEdit = hasPermission(user, 'invoice.update') && isDraft;
  const canPay =
    hasPermission(user, 'payment.create') &&
    (invoice.status === 'unpaid' || invoice.status === 'partial') &&
    invoice.remaining_amount > 0;
  const canVoidInvoice =
    hasPermission(user, 'invoice.void') &&
    !isVoid &&
    invoice.paid_amount === 0 &&
    !invoice.payments.some((payment) => payment.status === 'posted');
  const canVoidPayment = hasPermission(user, 'payment.void');
  const phoneDigits = invoice.customer_phone.replace(/\D/g, '');
  const paidPercent =
    invoice.total_amount > 0
      ? Math.min(100, Math.round((invoice.paid_amount / invoice.total_amount) * 100))
      : 100;

  return (
    <div className="space-y-6 pb-12">
      <div className="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
        <div className="flex items-start gap-3">
          <Button
            variant="outline"
            size="sm"
            onClick={() => navigate('/invoices')}
            className="mt-0.5 h-9 w-9 rounded-full p-0"
            title="Kembali"
          >
            <ArrowLeft className="h-4 w-4" aria-hidden="true" />
          </Button>
          <div>
            <div className="flex flex-wrap items-center gap-2">
              <h1 className="font-mono text-2xl font-bold tracking-tight text-ink sm:text-[28px]">
                {invoice.invoice_no}
              </h1>
              <InvoiceStatusBadge status={invoice.status} />
            </div>
            <p className="mt-1 text-sm text-muted">
              {invoice.customer_name} · Tiket{' '}
              <Link to={`/services/${invoice.service_order_id}`} className="font-medium text-accent-600 hover:underline">
                {invoice.ticket_no}
              </Link>{' '}
              · {formatDateTime(invoice.issued_at ?? invoice.created_at)}
            </p>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {canEdit && (
            <Button variant="outline" size="sm" onClick={() => setDiscountOpen(true)} className="gap-1.5">
              <BadgePercent className="h-4 w-4" aria-hidden="true" />
              Diskon
            </Button>
          )}
          {canIssue && (
            <Button size="sm" onClick={handleIssue} loading={issuing} className="gap-1.5">
              <CheckCircle2 className="h-4 w-4" aria-hidden="true" />
              Finalisasi
            </Button>
          )}
          {canPay && (
            <Button size="sm" onClick={() => setPaymentOpen(true)} className="gap-1.5">
              <Banknote className="h-4 w-4" aria-hidden="true" />
              Terima Pembayaran
            </Button>
          )}
          <Button variant="outline" size="sm" onClick={() => openPrint('a4')} className="gap-1.5">
            <Printer className="h-4 w-4" aria-hidden="true" />
            Cetak
          </Button>
          {phoneDigits.length > 0 && (
            <a href={whatsappInvoiceLink(invoice.customer_phone, invoice)} target="_blank" rel="noopener noreferrer">
              <Button variant="outline" size="sm" className="gap-1.5 text-emerald-700 hover:text-emerald-800">
                <MessageCircle className="h-4 w-4" aria-hidden="true" />
                WhatsApp
              </Button>
            </a>
          )}
          {canVoidInvoice && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => setVoidInvoiceOpen(true)}
              className="gap-1.5 border-red-200 text-danger hover:bg-red-50"
            >
              <Ban className="h-4 w-4" aria-hidden="true" />
              Batalkan
            </Button>
          )}
        </div>
      </div>

      {success && (
        <Alert variant="success">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <span>{success}</span>
            {invoice.payments.some((payment) => payment.status === 'posted') && (
              <Button variant="outline" size="sm" onClick={() => openPrint('receipt')} className="gap-1.5">
                <ReceiptText className="h-3.5 w-3.5" aria-hidden="true" />
                Cetak Struk
              </Button>
            )}
          </div>
        </Alert>
      )}

      {error && <Alert variant="error">{error}</Alert>}

      {isVoid && (
        <Alert variant="error">
          Invoice dibatalkan{invoice.voided_by ? ` oleh ${invoice.voided_by}` : ''}
          {invoice.voided_at ? ` pada ${formatDateTime(invoice.voided_at)}` : ''}.
          {invoice.void_reason ? ` Alasan: ${invoice.void_reason}` : ''}
        </Alert>
      )}

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <div className="space-y-6 lg:col-span-2">
          <Card>
            <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-2 pb-3">
              <div>
                <CardTitle className="text-base font-semibold">Rincian Tagihan</CardTitle>
                <p className="text-xs text-muted">
                  {invoice.customer_name} · {invoice.customer_phone}
                </p>
              </div>
              <div className="text-right text-xs text-muted">
                <p>
                  {invoice.service_order.device_brand} {invoice.service_order.device_model}
                  {invoice.service_order.device_color ? ` · ${invoice.service_order.device_color}` : ''}
                </p>
                {invoice.service_order.technician_name && <p>Teknisi: {invoice.service_order.technician_name}</p>}
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              {isDraft && (
                <Alert variant="info">
                  Invoice masih draft. Item snapshot belum dibuat; rincian item akan disalin dari item servis saat
                  finalisasi.{' '}
                  <Link to={`/services/${invoice.service_order_id}`} className="font-semibold underline">
                    Periksa item servis
                  </Link>
                  .
                </Alert>
              )}

              <div className="overflow-x-auto">
                <table className="w-full text-left text-sm">
                  <thead className="border-b border-border text-xs uppercase tracking-wider text-muted">
                    <tr>
                      <th scope="col" className="py-2 pr-3 font-semibold">Item</th>
                      <th scope="col" className="px-2 py-2 text-center font-semibold">Qty</th>
                      <th scope="col" className="px-2 py-2 text-right font-semibold">Harga</th>
                      <th scope="col" className="px-2 py-2 text-right font-semibold">Diskon</th>
                      <th scope="col" className="py-2 pl-3 text-right font-semibold">Subtotal</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-border">
                    {invoice.items.length === 0 ? (
                      <tr>
                        <td colSpan={5}>
                          <EmptyState
                            icon={Wallet}
                            title="Belum ada item invoice"
                            description="Finalisasi invoice untuk menyalin item servis dan sparepart sebagai snapshot permanen."
                            className="py-8"
                          />
                        </td>
                      </tr>
                    ) : (
                      invoice.items.map((item) => (
                        <tr key={item.id}>
                          <td className="py-3 pr-3">
                            <p className="font-medium text-ink">{item.description}</p>
                            <p className="text-[11px] uppercase tracking-wide text-muted">
                              {itemTypeLabel(item.item_type)}
                              {item.warranty_days > 0 ? ` · Garansi ${item.warranty_days} hari` : ''}
                            </p>
                          </td>
                          <td className="px-2 py-3 text-center font-mono text-xs">
                            {item.qty.toLocaleString('id-ID', { maximumFractionDigits: 2 })}
                          </td>
                          <td className="px-2 py-3 text-right font-mono text-xs">{formatCurrency(item.unit_price)}</td>
                          <td className="px-2 py-3 text-right font-mono text-xs">
                            {item.discount_amount > 0 ? `- ${formatCurrency(item.discount_amount)}` : '—'}
                          </td>
                          <td className="py-3 pl-3 text-right font-mono text-xs font-semibold">
                            {formatCurrency(item.subtotal)}
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>

              <div className="ml-auto w-full max-w-sm space-y-1.5 rounded-control border border-border bg-background p-4 text-sm">
                <div className="flex justify-between text-xs">
                  <span className="text-muted">Subtotal jasa</span>
                  <span className="font-mono">{formatCurrency(invoice.subtotal_service)}</span>
                </div>
                <div className="flex justify-between text-xs">
                  <span className="text-muted">Subtotal sparepart</span>
                  <span className="font-mono">{formatCurrency(invoice.subtotal_part)}</span>
                </div>
                {invoice.discount_amount > 0 && (
                  <div className="flex justify-between text-xs">
                    <span className="text-muted">
                      Diskon{invoice.discount_type === 'percent' ? ` (${invoice.discount_value}%)` : ''}
                    </span>
                    <span className="font-mono text-danger">- {formatCurrency(invoice.discount_amount)}</span>
                  </div>
                )}
                <div className="flex justify-between text-xs">
                  <span className="text-muted">Pajak</span>
                  <span className="font-mono">{formatCurrency(invoice.tax_amount)}</span>
                </div>
                <div className="flex justify-between border-t border-border pt-2 font-semibold">
                  <span>Total</span>
                  <span className="font-mono">{formatCurrency(invoice.total_amount)}</span>
                </div>
                <div className="flex justify-between text-xs text-emerald-700">
                  <span>Dibayar</span>
                  <span className="font-mono">{formatCurrency(invoice.paid_amount)}</span>
                </div>
                <div className="flex justify-between text-sm font-bold text-amber-700">
                  <span>Sisa</span>
                  <span className="font-mono">{formatCurrency(invoice.remaining_amount)}</span>
                </div>
              </div>

              {invoice.notes && (
                <div className="rounded-control border border-border bg-background p-3">
                  <p className="text-xs font-semibold uppercase tracking-wide text-muted">Catatan Invoice</p>
                  <p className="mt-1 whitespace-pre-line text-sm text-ink">{invoice.notes}</p>
                </div>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-2 pb-3">
              <div>
                <CardTitle className="text-base font-semibold">Riwayat Pembayaran</CardTitle>
                <p className="text-xs text-muted">Semua pembayaran posted dan void tercatat pada invoice ini.</p>
              </div>
              {canPay && (
                <Button size="sm" onClick={() => setPaymentOpen(true)} className="gap-1.5">
                  <Banknote className="h-3.5 w-3.5" aria-hidden="true" />
                  Tambah Pembayaran
                </Button>
              )}
            </CardHeader>
            <CardContent>
              {invoice.payments.length === 0 ? (
                <EmptyState
                  icon={Banknote}
                  title="Belum ada pembayaran"
                  description={
                    isDraft
                      ? 'Finalisasi invoice terlebih dahulu sebelum menerima pembayaran.'
                      : 'Pembayaran tunai, transfer, QRIS, kartu, maupun DP akan tampil di sini.'
                  }
                  className="py-8"
                />
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full text-left text-sm">
                    <thead className="border-b border-border text-xs uppercase tracking-wider text-muted">
                      <tr>
                        <th scope="col" className="py-2 pr-3 font-semibold">Pembayaran</th>
                        <th scope="col" className="px-2 py-2 font-semibold">Metode</th>
                        <th scope="col" className="px-2 py-2 text-right font-semibold">Nominal</th>
                        <th scope="col" className="px-2 py-2 font-semibold">Waktu</th>
                        <th scope="col" className="px-2 py-2 font-semibold">Status</th>
                        <th scope="col" className="py-2 pl-3 text-right font-semibold">Aksi</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-border">
                      {invoice.payments.map((payment) => {
                        const statusConfig = PAYMENT_STATUS[payment.status];
                        return (
                          <tr key={payment.id}>
                            <td className="py-3 pr-3">
                              <p className="font-mono text-xs font-semibold text-ink">{payment.payment_no}</p>
                              {payment.received_by && <p className="text-[11px] text-muted">Kasir: {payment.received_by}</p>}
                              {payment.reference_no && (
                                <p className="text-[11px] text-muted">Ref: {payment.reference_no}</p>
                              )}
                              {payment.notes && <p className="text-[11px] italic text-muted">{payment.notes}</p>}
                              {payment.status === 'void' && payment.void_reason && (
                                <p className="text-[11px] text-danger">Void: {payment.void_reason}</p>
                              )}
                            </td>
                            <td className="px-2 py-3 text-xs">{paymentMethodLabel(payment.method)}</td>
                            <td className="px-2 py-3 text-right font-mono text-xs font-semibold">
                              {formatCurrency(payment.amount)}
                            </td>
                            <td className="px-2 py-3 text-xs text-muted">{formatDateTime(payment.paid_at)}</td>
                            <td className="px-2 py-3">
                              <Badge variant={statusConfig?.variant ?? 'neutral'}>
                                {statusConfig?.label ?? payment.status}
                              </Badge>
                            </td>
                            <td className="py-3 pl-3 text-right">
                              {canVoidPayment && payment.status === 'posted' ? (
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  onClick={() => setVoidPaymentTarget(payment)}
                                  className="h-8 px-2 text-xs text-danger hover:bg-red-50"
                                >
                                  Void
                                </Button>
                              ) : (
                                <span className="text-xs text-muted">—</span>
                              )}
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        <div className="space-y-6">
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-semibold uppercase tracking-wider text-muted">
                Ringkasan Pembayaran
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <p className="text-xs text-muted">Sisa Tagihan</p>
                <p className="text-2xl font-bold font-mono text-amber-700">{formatCurrency(invoice.remaining_amount)}</p>
              </div>
              <div>
                <div className="mb-1 flex justify-between text-xs text-muted">
                  <span>Terbayar {paidPercent}%</span>
                  <span>{formatCurrency(invoice.paid_amount)}</span>
                </div>
                <div className="h-2 overflow-hidden rounded-full bg-background">
                  <div
                    className={`h-full rounded-full ${paidPercent >= 100 ? 'bg-emerald-500' : 'bg-accent-500'}`}
                    style={{ width: `${paidPercent}%` }}
                  />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-3 border-t border-border pt-3 text-xs">
                <div>
                  <p className="text-muted">Status</p>
                  <p className="mt-0.5 font-semibold text-ink">{invoiceStatusLabel(invoice.status)}</p>
                </div>
                <div>
                  <p className="text-muted">Total</p>
                  <p className="mt-0.5 font-mono font-semibold text-ink">{formatCurrency(invoice.total_amount)}</p>
                </div>
              </div>
              {canPay && (
                <Button onClick={() => setPaymentOpen(true)} className="w-full gap-1.5">
                  <Banknote className="h-4 w-4" aria-hidden="true" />
                  Terima Pembayaran
                </Button>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-semibold">Informasi Servis</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2 text-xs">
              <div className="flex justify-between gap-3">
                <span className="text-muted">Tiket</span>
                <Link
                  to={`/services/${invoice.service_order_id}`}
                  className="font-mono font-semibold text-accent-600 hover:underline"
                >
                  {invoice.ticket_no}
                </Link>
              </div>
              <div className="flex justify-between gap-3">
                <span className="text-muted">Status servis</span>
                <span className="font-medium text-ink">{invoice.service_order.status}</span>
              </div>
              <div className="flex justify-between gap-3">
                <span className="text-muted">Outlet</span>
                <span className="text-right font-medium text-ink">{invoice.outlet_name}</span>
              </div>
              <div className="flex justify-between gap-3">
                <span className="text-muted">Diterima</span>
                <span className="text-right text-ink">{formatDateTime(invoice.service_order.received_at)}</span>
              </div>
              {invoice.service_order.completed_at && (
                <div className="flex justify-between gap-3">
                  <span className="text-muted">Selesai</span>
                  <span className="text-right text-ink">{formatDateTime(invoice.service_order.completed_at)}</span>
                </div>
              )}
              {invoice.service_order.picked_up_at && (
                <div className="flex justify-between gap-3">
                  <span className="text-muted">Diambil</span>
                  <span className="text-right text-ink">{formatDateTime(invoice.service_order.picked_up_at)}</span>
                </div>
              )}
              <div className="border-t border-border pt-2">
                <Link to={`/services/${invoice.service_order_id}`}>
                  <Button variant="outline" size="sm" className="w-full">
                    Buka Detail Servis
                  </Button>
                </Link>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-semibold">Informasi Invoice</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2 text-xs">
              <div className="flex justify-between gap-3">
                <span className="text-muted">Dibuat</span>
                <span className="text-right text-ink">{formatDateTime(invoice.created_at)}</span>
              </div>
              <div className="flex justify-between gap-3">
                <span className="text-muted">Finalisasi</span>
                <span className="text-right text-ink">{formatDate(invoice.issued_at)}</span>
              </div>
              <div className="flex justify-between gap-3">
                <span className="text-muted">Jatuh tempo</span>
                <span className="text-right text-ink">{formatDate(invoice.due_at)}</span>
              </div>
              <div className="flex justify-between gap-3">
                <span className="text-muted">Tipe diskon</span>
                <span className="text-right text-ink">
                  {invoice.discount_type === 'percent' ? `${invoice.discount_value}%` : formatCurrency(invoice.discount_value)}
                </span>
              </div>
              <div className="border-t border-border pt-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => openPrint('receipt')}
                  className="w-full gap-1.5"
                >
                  <ReceiptText className="h-3.5 w-3.5" aria-hidden="true" />
                  Cetak Struk Thermal
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>

      <PaymentDialog
        open={paymentOpen}
        onOpenChange={setPaymentOpen}
        invoice={invoice}
        onSuccess={handlePaymentSuccess}
      />

      <InvoiceDiscountDialog
        open={discountOpen}
        onOpenChange={setDiscountOpen}
        invoice={invoice}
        onSuccess={(updated) => {
          setInvoice(updated);
          setSuccess('Diskon dan catatan invoice berhasil diperbarui.');
        }}
      />

      <VoidReasonDialog
        open={voidInvoiceOpen}
        onOpenChange={setVoidInvoiceOpen}
        title="Batalkan Invoice"
        description={`Invoice ${invoice.invoice_no} akan dibatalkan. Tindakan ini hanya dapat dilakukan jika belum ada pembayaran posted.`}
        confirmLabel="Batalkan Invoice"
        onSubmit={handleVoidInvoice}
      />

      {voidPaymentTarget && (
        <VoidReasonDialog
          open
          onOpenChange={(open) => {
            if (!open) setVoidPaymentTarget(null);
          }}
          title="Void Pembayaran"
          description={`Pembayaran ${voidPaymentTarget.payment_no} sebesar ${formatCurrency(
            voidPaymentTarget.amount,
          )} akan dibatalkan dan status invoice dihitung ulang.`}
          confirmLabel="Void Pembayaran"
          onSubmit={handleVoidPayment}
        />
      )}

      <PrintInvoiceModal
        open={printOpen}
        onOpenChange={setPrintOpen}
        invoiceId={invoice.id}
        initialTemplate={printTemplate}
      />
    </div>
  );
}
