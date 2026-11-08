import { api, apiRequest } from '@/lib/api-client';
import type { ApiEnvelope } from '@/types/api';
import { invoiceStatusLabel } from './format';
import type {
  CreatePaymentRequest,
  DailySummary,
  InvoiceDetail,
  InvoiceFilter,
  InvoiceListItem,
  InvoicePayment,
  InvoicePrintPayload,
  PaymentFilter,
  PaymentListItem,
  PaymentMutation,
  UpdateInvoiceRequest,
} from './types';

function pageQuery(filter: object): string {
  const params = new URLSearchParams();
  Object.entries(filter).forEach(([key, value]) => {
    if (value !== undefined && value !== null && String(value).trim() !== '') {
      params.set(key, String(value));
    }
  });
  const query = params.toString();
  return query ? `?${query}` : '';
}

export async function listInvoices(filter: InvoiceFilter = {}): Promise<ApiEnvelope<InvoiceListItem[]>> {
  return api.getEnvelope<InvoiceListItem[]>(`/invoices${pageQuery(filter)}`);
}

export async function getInvoice(id: string): Promise<InvoiceDetail> {
  return api.get<InvoiceDetail>(`/invoices/${id}`);
}

export async function getInvoiceByServiceOrder(orderId: string): Promise<InvoiceDetail> {
  return api.get<InvoiceDetail>(`/service-orders/${orderId}/invoice`);
}

export async function updateInvoice(id: string, payload: UpdateInvoiceRequest): Promise<InvoiceDetail> {
  return api.patch<InvoiceDetail>(`/invoices/${id}`, payload);
}

export async function getInvoicePrint(id: string): Promise<InvoicePrintPayload> {
  return api.get<InvoicePrintPayload>(`/invoices/${id}/print`);
}

export async function issueInvoice(id: string): Promise<InvoiceDetail> {
  return api.post<InvoiceDetail>(`/invoices/${id}/issue`);
}

export async function voidInvoice(id: string, reason: string): Promise<InvoiceDetail> {
  return api.post<InvoiceDetail>(`/invoices/${id}/void`, { reason });
}

export async function listInvoicePayments(invoiceId: string): Promise<InvoicePayment[]> {
  return api.get<InvoicePayment[]>(`/invoices/${invoiceId}/payments`);
}

export async function listPayments(filter: PaymentFilter = {}): Promise<ApiEnvelope<PaymentListItem[]>> {
  return api.getEnvelope<PaymentListItem[]>(`/payments${pageQuery(filter)}`);
}

export async function getDailySummary(date?: string, outletId?: string): Promise<DailySummary> {
  return api.get<DailySummary>(`/payments/daily-summary${pageQuery({ date, outlet_id: outletId })}`);
}

export async function createPayment(
  invoiceId: string,
  payload: CreatePaymentRequest,
  idempotencyKey: string,
): Promise<PaymentMutation> {
  return apiRequest<PaymentMutation>(`/invoices/${invoiceId}/payments`, {
    method: 'POST',
    body: payload,
    headers: { 'Idempotency-Key': idempotencyKey },
  });
}

export async function voidPayment(paymentId: string, reason: string): Promise<PaymentMutation> {
  return api.post<PaymentMutation>(`/payments/${paymentId}/void`, { reason });
}

export function newIdempotencyKey(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  return `pay-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
}

export function whatsappInvoiceLink(phone: string, invoice: InvoiceDetail): string {
  const normalized = phone.replace(/\D/g, '').replace(/^0/, '62');
  const lines = [
    `Halo ${invoice.customer_name},`,
    `Invoice ${invoice.invoice_no} untuk tiket ${invoice.ticket_no}.`,
    `Total: ${new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(invoice.total_amount)}`,
    `Dibayar: ${new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(invoice.paid_amount)}`,
    `Sisa: ${new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(invoice.remaining_amount)}`,
    `Status: ${invoiceStatusLabel(invoice.status)}`,
  ];
  if (invoice.store?.name) {
    lines.push(`Terima kasih - ${invoice.store.name}`);
  }
  return `https://wa.me/${normalized}?text=${encodeURIComponent(lines.join('\n'))}`;
}
