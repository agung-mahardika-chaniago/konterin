import * as React from 'react';
import { Link } from 'react-router-dom';
import {
  Banknote,
  Ban,
  CalendarDays,
  ChevronLeft,
  ChevronRight,
  CreditCard,
  Eye,
  QrCode,
  ReceiptText,
  RefreshCw,
  Search,
  Wallet,
} from 'lucide-react';

import { EmptyState } from '@/components/shared/empty-state';
import { Alert } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Select } from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { formatCurrency } from '@/features/dashboard/format';
import { PrintInvoiceModal } from '@/features/print/print-invoice-modal';
import type { PrintTemplate } from '@/features/print/print-invoice-modal';
import { ApiError } from '@/lib/api-client';
import { hasPermission } from '@/lib/permissions';
import { useAuthStore } from '@/stores/auth-store';
import {
  getDailySummary,
  getInvoice,
  listInvoices,
  listPayments,
  voidPayment,
} from '../api';
import { InvoiceStatusBadge } from '../components/invoice-status-badge';
import { PaymentDialog } from '../components/payment-dialog';
import { VoidReasonDialog } from '../components/void-reason-dialog';
import {
  PAYMENT_METHOD,
  PAYMENT_STATUS,
  formatDate,
  formatDateTime,
  paymentMethodLabel,
  todayInput,
} from '../format';
import type {
  DailySummary,
  InvoiceDetail,
  InvoiceListItem,
  PaymentListItem,
  PaymentMethod,
  PaymentStatus,
} from '../types';

const METHOD_OPTIONS = Object.keys(PAYMENT_METHOD) as PaymentMethod[];
const STATUS_OPTIONS: PaymentStatus[] = ['posted', 'void'];

export function CashierPage() {
  const user = useAuthStore((state) => state.user);
  const canPay = hasPermission(user, 'payment.create');
  const canVoid = hasPermission(user, 'payment.void');
  const canViewInvoice = hasPermission(user, 'invoice.view');

  const [date, setDate] = React.useState(todayInput());
  const [method, setMethod] = React.useState('');
  const [status, setStatus] = React.useState('');
  const [search, setSearch] = React.useState('');
  const [debouncedSearch, setDebouncedSearch] = React.useState('');
  const [page, setPage] = React.useState(1);

  const [payments, setPayments] = React.useState<PaymentListItem[]>([]);
  const [total, setTotal] = React.useState(0);
  const [totalPages, setTotalPages] = React.useState(1);
  const [paymentsLoading, setPaymentsLoading] = React.useState(true);
  const [paymentsError, setPaymentsError] = React.useState<string | null>(null);

  const [summary, setSummary] = React.useState<DailySummary | null>(null);
  const [summaryLoading, setSummaryLoading] = React.useState(true);
  const [summaryError, setSummaryError] = React.useState<string | null>(null);

  const [outstanding, setOutstanding] = React.useState<InvoiceListItem[]>([]);
  const [outstandingLoading, setOutstandingLoading] = React.useState(true);
  const [outstandingError, setOutstandingError] = React.useState<string | null>(null);

  const [payingInvoice, setPayingInvoice] = React.useState<InvoiceDetail | null>(null);
  const [loadingInvoiceId, setLoadingInvoiceId] = React.useState<string | null>(null);
  const [voidTarget, setVoidTarget] = React.useState<PaymentListItem | null>(null);
  const [printOpen, setPrintOpen] = React.useState(false);
  const [printInvoiceId, setPrintInvoiceId] = React.useState<string | null>(null);
  const [printTemplate, setPrintTemplate] = React.useState<PrintTemplate>('receipt');
  const [success, setSuccess] = React.useState<string | null>(null);
  const [actionError, setActionError] = React.useState<string | null>(null);

  React.useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search);
      setPage(1);
    }, 300);
    return () => clearTimeout(timer);
  }, [search]);

  const loadSummary = React.useCallback(async () => {
    setSummaryLoading(true);
    setSummaryError(null);
    try {
      const data = await getDailySummary(date);
      setSummary(data);
    } catch (err: unknown) {
      setSummaryError(err instanceof Error ? err.message : 'Gagal memuat ringkasan kasir.');
    } finally {
      setSummaryLoading(false);
    }
  }, [date]);

  const loadPayments = React.useCallback(async () => {
    setPaymentsLoading(true);
    setPaymentsError(null);
    try {
      const result = await listPayments({
        q: debouncedSearch.trim() || undefined,
        method: method || undefined,
        status: status || undefined,
        from: date,
        to: date,
        page,
        per_page: 15,
      });
      setPayments(result.data ?? []);
      if (result.meta) {
        setTotal(result.meta.total);
        setTotalPages(result.meta.total_pages || 1);
      }
    } catch (err: unknown) {
      setPaymentsError(err instanceof Error ? err.message : 'Gagal memuat daftar transaksi.');
    } finally {
      setPaymentsLoading(false);
    }
  }, [debouncedSearch, method, status, date, page]);

  const loadOutstanding = React.useCallback(async () => {
    if (!canViewInvoice) {
      setOutstandingLoading(false);
      return;
    }
    setOutstandingLoading(true);
    setOutstandingError(null);
    try {
      const result = await listInvoices({
        status: 'unpaid,partial',
        page: 1,
        per_page: 8,
      });
      setOutstanding(result.data ?? []);
    } catch (err: unknown) {
      setOutstandingError(err instanceof Error ? err.message : 'Gagal memuat tagihan terbuka.');
    } finally {
      setOutstandingLoading(false);
    }
  }, [canViewInvoice]);

  React.useEffect(() => {
    loadSummary();
  }, [loadSummary]);

  React.useEffect(() => {
    loadPayments();
  }, [loadPayments]);

  React.useEffect(() => {
    loadOutstanding();
  }, [loadOutstanding]);

  const refreshAll = () => {
    loadSummary();
    loadPayments();
    loadOutstanding();
  };

  const handleOpenPayment = async (invoiceId: string) => {
    setLoadingInvoiceId(invoiceId);
    setActionError(null);
    try {
      const detail = await getInvoice(invoiceId);
      setPayingInvoice(detail);
    } catch (err: unknown) {
      setActionError(err instanceof ApiError ? err.message : 'Gagal memuat invoice.');
    } finally {
      setLoadingInvoiceId(null);
    }
  };

  const handlePaymentSuccess = (result: { payment: { amount: number; payment_no: string }; invoice: { id: string } }) => {
    setSuccess(`Pembayaran ${result.payment.payment_no} sebesar ${formatCurrency(result.payment.amount)} berhasil dicatat.`);
    setPrintInvoiceId(result.invoice.id);
    refreshAll();
  };

  const handleVoidPayment = async (reason: string) => {
    if (!voidTarget) return;
    const result = await voidPayment(voidTarget.id, reason);
    setSuccess(`Pembayaran ${result.payment.payment_no} berhasil dibatalkan.`);
    refreshAll();
  };

  const openPrint = (invoiceId: string, template: PrintTemplate) => {
    setPrintInvoiceId(invoiceId);
    setPrintTemplate(template);
    setPrintOpen(true);
  };

  const resetFilters = () => {
    setSearch('');
    setMethod('');
    setStatus('');
    setPage(1);
  };

  const hasFilter = Boolean(search || method || status);
  const paymentTotal = payments.reduce((sum, payment) => {
    return payment.status === 'posted' ? sum + payment.amount : sum;
  }, 0);

  return (
    <div className="space-y-5 pb-12">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-ink sm:text-[28px]">Kasir</h1>
          <p className="mt-1 text-sm text-muted">
            Ringkasan penerimaan harian, rekap metode bayar, dan transaksi yang tercatat.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <div className="relative">
            <CalendarDays
              className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted"
              aria-hidden="true"
            />
            <Input
              type="date"
              value={date}
              onChange={(event) => {
                setDate(event.target.value || todayInput());
                setPage(1);
              }}
              className="w-[175px] pl-9"
              title="Tanggal laporan"
            />
          </div>
          <Button variant="outline" onClick={refreshAll} className="gap-1.5">
            <RefreshCw className="h-4 w-4" aria-hidden="true" />
            Muat Ulang
          </Button>
        </div>
      </div>

      {success && <Alert variant="success">{success}</Alert>}
      {actionError && <Alert variant="error">{actionError}</Alert>}
      {summaryError && <Alert variant="error">{summaryError}</Alert>}

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {summaryLoading ? (
          [1, 2, 3, 4].map((i) => <Skeleton key={i} className="h-24 w-full" />)
        ) : (
          <>
            <Card className="p-4">
              <div className="flex items-center justify-between">
                <p className="text-xs font-medium uppercase tracking-wide text-muted">Pemasukan</p>
                <Banknote className="h-4 w-4 text-emerald-600" aria-hidden="true" />
              </div>
              <p className="mt-1 text-2xl font-bold text-ink">{formatCurrency(summary?.total_amount ?? 0)}</p>
              <p className="text-xs text-muted">{summary?.posted_count ?? 0} pembayaran posted</p>
            </Card>
            <Card className="p-4">
              <div className="flex items-center justify-between">
                <p className="text-xs font-medium uppercase tracking-wide text-muted">Transaksi</p>
                <ReceiptText className="h-4 w-4 text-accent-600" aria-hidden="true" />
              </div>
              <p className="mt-1 text-2xl font-bold text-ink">{summary?.posted_count ?? 0}</p>
              <p className="text-xs text-muted">{summary?.invoice_count ?? 0} invoice terbayar</p>
            </Card>
            <Card className="p-4">
              <div className="flex items-center justify-between">
                <p className="text-xs font-medium uppercase tracking-wide text-muted">Invoice Dibayar</p>
                <Wallet className="h-4 w-4 text-accent-600" aria-hidden="true" />
              </div>
              <p className="mt-1 text-2xl font-bold text-ink">{summary?.invoice_count ?? 0}</p>
              <p className="text-xs text-muted">{formatDate(summary?.date ?? date)}</p>
            </Card>
            <Card className="p-4">
              <div className="flex items-center justify-between">
                <p className="text-xs font-medium uppercase tracking-wide text-muted">Void</p>
                <Ban className="h-4 w-4 text-danger" aria-hidden="true" />
              </div>
              <p className="mt-1 text-2xl font-bold text-ink">{summary?.voided_count ?? 0}</p>
              <p className="text-xs text-muted">{formatCurrency(summary?.voided_amount ?? 0)}</p>
            </Card>
          </>
        )}
      </div>

      <Card>
        <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-2 pb-3">
          <div>
            <CardTitle className="text-base font-semibold">Transaksi Tanggal {formatDate(date)}</CardTitle>
            <p className="text-xs text-muted">
              Menampilkan {payments.length} dari {total} transaksi · nilai posted halaman ini{' '}
              {formatCurrency(paymentTotal)}.
            </p>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {paymentsError && <Alert variant="error">{paymentsError}</Alert>}

          <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
            <div className="relative flex-1 sm:max-w-sm">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted" aria-hidden="true" />
              <Input
                value={search}
                onChange={(event) => setSearch(event.target.value)}
                placeholder="Cari nomor pembayaran, invoice, tiket, pelanggan..."
                className="pl-9"
              />
            </div>
            <Select
              value={method}
              onChange={(event) => {
                setMethod(event.target.value);
                setPage(1);
              }}
              className="sm:max-w-[180px]"
            >
              <option value="">Semua metode</option>
              {METHOD_OPTIONS.map((value) => (
                <option key={value} value={value}>
                  {PAYMENT_METHOD[value].label}
                </option>
              ))}
            </Select>
            <Select
              value={status}
              onChange={(event) => {
                setStatus(event.target.value);
                setPage(1);
              }}
              className="sm:max-w-[150px]"
            >
              <option value="">Semua status</option>
              {STATUS_OPTIONS.map((value) => (
                <option key={value} value={value}>
                  {PAYMENT_STATUS[value].label}
                </option>
              ))}
            </Select>
            {hasFilter && (
              <Button variant="ghost" size="sm" onClick={resetFilters}>
                Reset
              </Button>
            )}
          </div>

          {paymentsLoading ? (
            <div className="space-y-3">
              {[1, 2, 3, 4].map((i) => (
                <div key={i} className="flex items-center justify-between py-2">
                  <div className="space-y-1">
                    <Skeleton className="h-4 w-48" />
                    <Skeleton className="h-3 w-32" />
                  </div>
                  <Skeleton className="h-8 w-24" />
                </div>
              ))}
            </div>
          ) : payments.length === 0 ? (
            <EmptyState
              icon={ReceiptText}
              title="Belum ada transaksi"
              description={
                hasFilter
                  ? 'Tidak ada transaksi yang sesuai dengan filter Anda pada tanggal ini.'
                  : 'Pembayaran yang diterima kasir akan muncul di sini secara otomatis.'
              }
              className="py-10"
            />
          ) : (
            <>
              <div className="overflow-x-auto">
                <table className="w-full text-left text-sm">
                  <thead className="border-b border-border text-xs uppercase tracking-wider text-muted">
                    <tr>
                      <th scope="col" className="py-2 pr-3 font-semibold">Pembayaran</th>
                      <th scope="col" className="px-2 py-2 font-semibold">Pelanggan</th>
                      <th scope="col" className="px-2 py-2 font-semibold">Metode</th>
                      <th scope="col" className="px-2 py-2 text-right font-semibold">Nominal</th>
                      <th scope="col" className="px-2 py-2 font-semibold">Waktu</th>
                      <th scope="col" className="px-2 py-2 font-semibold">Status</th>
                      <th scope="col" className="py-2 pl-3 text-right font-semibold">Aksi</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-border">
                    {payments.map((payment) => {
                      const statusConfig = PAYMENT_STATUS[payment.status];
                      return (
                        <tr key={payment.id}>
                          <td className="py-3 pr-3">
                            <p className="font-mono text-xs font-semibold text-ink">{payment.payment_no}</p>
                            <Link
                              to={`/invoices/${payment.invoice_id}`}
                              className="text-[11px] font-medium text-accent-600 hover:underline"
                            >
                              {payment.invoice_no}
                            </Link>
                            <span className="text-[11px] text-muted"> · {payment.ticket_no}</span>
                            {payment.reference_no && (
                              <p className="text-[11px] text-muted">Ref: {payment.reference_no}</p>
                            )}
                          </td>
                          <td className="px-2 py-3">
                            <p className="text-xs font-medium text-ink">{payment.customer_name}</p>
                            <p className="text-[11px] text-muted">{payment.customer_phone}</p>
                          </td>
                          <td className="px-2 py-3 text-xs">{paymentMethodLabel(payment.method)}</td>
                          <td className="px-2 py-3 text-right font-mono text-xs font-semibold">
                            {formatCurrency(payment.amount)}
                          </td>
                          <td className="px-2 py-3 text-xs text-muted">{formatDateTime(payment.paid_at)}</td>
                          <td className="px-2 py-3">
                            <span
                              className={`inline-flex rounded-full px-2 py-0.5 text-[10px] font-semibold ring-1 ring-inset ${
                                statusConfig?.variant === 'danger'
                                  ? 'bg-red-50 text-red-700 ring-red-200'
                                  : 'bg-emerald-50 text-emerald-700 ring-emerald-200'
                              }`}
                            >
                              {statusConfig?.label ?? payment.status}
                            </span>
                            {payment.received_by && (
                              <p className="mt-0.5 text-[10px] text-muted">{payment.received_by}</p>
                            )}
                          </td>
                          <td className="py-3 pl-3 text-right">
                            <div className="flex items-center justify-end gap-1">
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => openPrint(payment.invoice_id, 'receipt')}
                                className="h-8 px-2"
                                title="Cetak struk"
                              >
                                <ReceiptText className="h-3.5 w-3.5" aria-hidden="true" />
                              </Button>
                              <Link to={`/invoices/${payment.invoice_id}`}>
                                <Button variant="ghost" size="sm" className="h-8 px-2" title="Buka invoice">
                                  <Eye className="h-3.5 w-3.5" aria-hidden="true" />
                                </Button>
                              </Link>
                              {canVoid && payment.status === 'posted' && (
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  onClick={() => setVoidTarget(payment)}
                                  className="h-8 px-2 text-xs text-danger hover:bg-red-50"
                                  title="Void pembayaran"
                                >
                                  Void
                                </Button>
                              )}
                            </div>
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>

              <div className="flex items-center justify-between border-t border-border pt-3 text-xs text-muted">
                <span>
                  Halaman {page} dari {totalPages}
                </span>
                <div className="flex items-center gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setPage((current) => Math.max(1, current - 1))}
                    disabled={page <= 1}
                    className="h-8 px-2"
                  >
                    <ChevronLeft className="h-4 w-4" aria-hidden="true" />
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setPage((current) => Math.min(totalPages, current + 1))}
                    disabled={page >= totalPages}
                    className="h-8 px-2"
                  >
                    <ChevronRight className="h-4 w-4" aria-hidden="true" />
                  </Button>
                </div>
              </div>
            </>
          )}
        </CardContent>
      </Card>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-base font-semibold">Rekap Metode Bayar</CardTitle>
            <p className="text-xs text-muted">Hanya pembayaran posted pada tanggal terpilih.</p>
          </CardHeader>
          <CardContent>
            {summaryLoading ? (
              <div className="space-y-3">
                {[1, 2, 3].map((i) => (
                  <Skeleton key={i} className="h-8 w-full" />
                ))}
              </div>
            ) : !summary || summary.methods.length === 0 ? (
              <EmptyState
                icon={CreditCard}
                title="Belum ada penerimaan"
                description="Rekap per metode akan muncul setelah kasir menerima pembayaran."
                className="py-8"
              />
            ) : (
              <div className="space-y-3">
                {summary.methods.map((item) => {
                  const percent =
                    summary.total_amount > 0 ? Math.round((item.amount / summary.total_amount) * 100) : 0;
                  const Icon =
                    item.method === 'cash'
                      ? Banknote
                      : item.method === 'qris'
                        ? QrCode
                        : item.method === 'card'
                          ? CreditCard
                          : Wallet;
                  return (
                    <div key={item.method}>
                      <div className="flex items-center justify-between gap-2 text-xs">
                        <span className="flex items-center gap-1.5 font-medium text-ink">
                          <Icon className="h-3.5 w-3.5 text-muted" aria-hidden="true" />
                          {paymentMethodLabel(item.method)}
                        </span>
                        <span className="font-mono text-ink">
                          {formatCurrency(item.amount)}{' '}
                          <span className="text-muted">({item.count}x)</span>
                        </span>
                      </div>
                      <div className="mt-1 h-1.5 overflow-hidden rounded-full bg-background">
                        <div className="h-full rounded-full bg-accent-500" style={{ width: `${percent}%` }} />
                      </div>
                    </div>
                  );
                })}
                <div className="flex justify-between border-t border-border pt-2 text-xs font-semibold">
                  <span>Total</span>
                  <span className="font-mono">{formatCurrency(summary.total_amount)}</span>
                </div>
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-3">
            <div>
              <CardTitle className="text-base font-semibold">Tagihan Belum Lunas</CardTitle>
              <p className="text-xs text-muted">Invoice unpaid/partial pada outlet aktif.</p>
            </div>
            {canViewInvoice && (
              <Link to="/invoices">
                <Button variant="ghost" size="sm" className="text-xs">
                  Semua invoice
                </Button>
              </Link>
            )}
          </CardHeader>
          <CardContent>
            {outstandingError && <Alert variant="error">{outstandingError}</Alert>}
            {outstandingLoading ? (
              <div className="space-y-3">
                {[1, 2, 3].map((i) => (
                  <Skeleton key={i} className="h-10 w-full" />
                ))}
              </div>
            ) : outstanding.length === 0 ? (
              <EmptyState
                icon={Wallet}
                title="Tidak ada tagihan terbuka"
                description="Semua invoice pada outlet ini sudah lunas atau belum difinalisasi."
                className="py-8"
              />
            ) : (
              <div className="divide-y divide-border">
                {outstanding.map((invoice) => (
                  <div key={invoice.id} className="flex flex-wrap items-center justify-between gap-2 py-2.5">
                    <div className="min-w-0">
                      <div className="flex items-center gap-2">
                        <Link
                          to={`/invoices/${invoice.id}`}
                          className="font-mono text-xs font-semibold text-ink hover:text-accent-600"
                        >
                          {invoice.invoice_no}
                        </Link>
                        <InvoiceStatusBadge status={invoice.status} />
                      </div>
                      <p className="text-[11px] text-muted">
                        {invoice.customer_name} · {invoice.ticket_no}
                      </p>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="font-mono text-xs font-semibold text-amber-700">
                        {formatCurrency(invoice.remaining_amount)}
                      </span>
                      {canPay ? (
                        <Button
                          size="sm"
                          onClick={() => handleOpenPayment(invoice.id)}
                          loading={loadingInvoiceId === invoice.id}
                          className="h-8 gap-1 px-2.5 text-xs"
                        >
                          <Banknote className="h-3.5 w-3.5" aria-hidden="true" />
                          Bayar
                        </Button>
                      ) : (
                        <Link to={`/invoices/${invoice.id}`}>
                          <Button variant="outline" size="sm" className="h-8 px-2.5 text-xs">
                            Detail
                          </Button>
                        </Link>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {payingInvoice && (
        <PaymentDialog
          open
          onOpenChange={(open) => {
            if (!open) setPayingInvoice(null);
          }}
          invoice={payingInvoice}
          onSuccess={handlePaymentSuccess}
        />
      )}

      {voidTarget && (
        <VoidReasonDialog
          open
          onOpenChange={(open) => {
            if (!open) setVoidTarget(null);
          }}
          title="Void Pembayaran"
          description={`Pembayaran ${voidTarget.payment_no} sebesar ${formatCurrency(
            voidTarget.amount,
          )} akan dibatalkan dan status invoice dihitung ulang.`}
          confirmLabel="Void Pembayaran"
          onSubmit={handleVoidPayment}
        />
      )}

      <PrintInvoiceModal
        open={printOpen}
        onOpenChange={setPrintOpen}
        invoiceId={printInvoiceId}
        initialTemplate={printTemplate}
      />
    </div>
  );
}
