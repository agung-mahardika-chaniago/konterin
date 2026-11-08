import { formatCurrency } from '@/features/dashboard/format';
import { formatDate, formatDateTime, invoiceStatusLabel, itemTypeLabel, paymentMethodLabel } from '../invoices/format';
import type { InvoiceDetail } from '../invoices/types';

function formatQty(value: number): string {
  return Number.isInteger(value) ? String(value) : value.toLocaleString('id-ID', { maximumFractionDigits: 2 });
}

export function InvoiceA4({ invoice }: { invoice: InvoiceDetail }) {
  const store = invoice.store;
  const isVoid = invoice.status === 'void';
  const isDraft = invoice.status === 'draft';
  const postedPayments = invoice.payments.filter((payment) => payment.status === 'posted');
  const logoSrc = store.logo_path ? new URL(store.logo_path, window.location.origin).href : null;

  return (
    <div className="relative w-[210mm] min-h-[297mm] overflow-hidden bg-white p-[12mm] text-[11px] leading-relaxed text-slate-900">
      {(isVoid || isDraft) && (
        <div className="pointer-events-none absolute inset-0 flex items-center justify-center">
          <span className="-rotate-[28deg] text-[110px] font-black uppercase tracking-[0.18em] text-slate-900/10">
            {isVoid ? 'VOID' : 'DRAFT'}
          </span>
        </div>
      )}

      <header className="relative flex items-start justify-between gap-8 border-b-2 border-slate-800 pb-5">
        <div className="flex min-w-0 items-start gap-3">
          {logoSrc ? (
            <img src={logoSrc} alt={store.name} className="h-14 w-14 rounded object-contain" />
          ) : (
            <span className="flex h-14 w-14 shrink-0 items-center justify-center rounded bg-slate-900 text-xl font-bold text-white">
              {store.name.charAt(0).toUpperCase()}
            </span>
          )}
          <div className="min-w-0">
            <p className="text-lg font-bold leading-tight">{store.name}</p>
            {store.legal_name && store.legal_name !== store.name && (
              <p className="text-[10px] text-slate-600">{store.legal_name}</p>
            )}
            {store.address && <p className="mt-1 whitespace-pre-line text-[10px] text-slate-600">{store.address}</p>}
            <p className="text-[10px] text-slate-600">
              {[store.phone, store.email].filter(Boolean).join(' · ')}
            </p>
            {store.tax_id && <p className="text-[10px] text-slate-600">NPWP: {store.tax_id}</p>}
          </div>
        </div>

        <div className="shrink-0 text-right">
          <p className="text-[22px] font-black uppercase tracking-[0.22em] text-slate-900">Invoice</p>
          <p className="mt-1 font-mono text-sm font-bold">{invoice.invoice_no}</p>
          <p className="mt-1 text-[10px] text-slate-600">Tanggal: {formatDate(invoice.issued_at ?? invoice.created_at)}</p>
          {invoice.due_at && <p className="text-[10px] text-slate-600">Jatuh tempo: {formatDate(invoice.due_at)}</p>}
          <span className="mt-2 inline-block rounded border border-slate-400 px-2 py-0.5 text-[9px] font-bold uppercase tracking-wide">
            {invoiceStatusLabel(invoice.status)}
          </span>
        </div>
      </header>

      <section className="relative mt-5 grid grid-cols-2 gap-5">
        <div className="rounded border border-slate-300 p-3">
          <p className="mb-1 text-[9px] font-bold uppercase tracking-[0.14em] text-slate-500">Ditagihkan kepada</p>
          <p className="text-sm font-bold">{invoice.customer_name}</p>
          <p className="text-[10px] text-slate-600">{invoice.customer_phone}</p>
          <p className="mt-1 text-[10px] text-slate-600">Outlet: {invoice.outlet_name}</p>
        </div>
        <div className="rounded border border-slate-300 p-3">
          <p className="mb-1 text-[9px] font-bold uppercase tracking-[0.14em] text-slate-500">Detail Servis</p>
          <p className="font-mono text-xs font-bold">{invoice.ticket_no}</p>
          <p className="text-[10px] text-slate-600">
            {invoice.service_order.device_brand} {invoice.service_order.device_model}
            {invoice.service_order.device_color ? ` · ${invoice.service_order.device_color}` : ''}
          </p>
          {invoice.service_order.device_imei && (
            <p className="text-[10px] text-slate-600">IMEI/SN: {invoice.service_order.device_imei}</p>
          )}
          {invoice.service_order.technician_name && (
            <p className="text-[10px] text-slate-600">Teknisi: {invoice.service_order.technician_name}</p>
          )}
        </div>
      </section>

      {invoice.service_order.complaint && (
        <section className="relative mt-4 rounded border border-slate-200 bg-slate-50 p-3">
          <p className="text-[9px] font-bold uppercase tracking-[0.14em] text-slate-500">Keluhan</p>
          <p className="text-[10px] text-slate-700">{invoice.service_order.complaint}</p>
        </section>
      )}

      <section className="relative mt-5">
        <table className="w-full border-collapse text-left">
          <thead>
            <tr className="border-y border-slate-800 bg-slate-100 text-[9px] uppercase tracking-wide text-slate-700">
              <th className="px-2 py-2 font-bold">Deskripsi</th>
              <th className="w-14 px-2 py-2 text-center font-bold">Qty</th>
              <th className="w-24 px-2 py-2 text-right font-bold">Harga</th>
              <th className="w-20 px-2 py-2 text-right font-bold">Diskon</th>
              <th className="w-28 px-2 py-2 text-right font-bold">Subtotal</th>
            </tr>
          </thead>
          <tbody>
            {invoice.items.length === 0 ? (
              <tr>
                <td colSpan={5} className="px-2 py-6 text-center text-[10px] italic text-slate-500">
                  Belum ada item snapshot. Item akan disimpan saat invoice difinalisasi.
                </td>
              </tr>
            ) : (
              invoice.items.map((item) => (
                <tr key={item.id} className="break-inside-avoid border-b border-slate-200 align-top">
                  <td className="px-2 py-2">
                    <p className="font-medium text-slate-900">{item.description}</p>
                    <p className="text-[9px] uppercase tracking-wide text-slate-500">{itemTypeLabel(item.item_type)}</p>
                    {item.warranty_days > 0 && (
                      <p className="text-[9px] text-slate-500">Garansi {item.warranty_days} hari</p>
                    )}
                  </td>
                  <td className="px-2 py-2 text-center font-mono">{formatQty(item.qty)}</td>
                  <td className="px-2 py-2 text-right font-mono">{formatCurrency(item.unit_price)}</td>
                  <td className="px-2 py-2 text-right font-mono">
                    {item.discount_amount > 0 ? `- ${formatCurrency(item.discount_amount)}` : '—'}
                  </td>
                  <td className="px-2 py-2 text-right font-mono font-semibold">{formatCurrency(item.subtotal)}</td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </section>

      <section className="relative mt-4 flex justify-end">
        <div className="w-[78mm] space-y-1 rounded border border-slate-300 p-3">
          <div className="flex justify-between text-[10px]">
            <span className="text-slate-600">Subtotal jasa</span>
            <span className="font-mono">{formatCurrency(invoice.subtotal_service)}</span>
          </div>
          <div className="flex justify-between text-[10px]">
            <span className="text-slate-600">Subtotal sparepart</span>
            <span className="font-mono">{formatCurrency(invoice.subtotal_part)}</span>
          </div>
          {invoice.discount_amount > 0 && (
            <div className="flex justify-between text-[10px]">
              <span className="text-slate-600">
                Diskon{invoice.discount_type === 'percent' ? ` (${invoice.discount_value}%)` : ''}
              </span>
              <span className="font-mono">- {formatCurrency(invoice.discount_amount)}</span>
            </div>
          )}
          <div className="flex justify-between text-[10px]">
            <span className="text-slate-600">Pajak</span>
            <span className="font-mono">{formatCurrency(invoice.tax_amount)}</span>
          </div>
          <div className="flex justify-between border-t border-slate-800 pt-1 text-sm font-bold">
            <span>Total</span>
            <span className="font-mono">{formatCurrency(invoice.total_amount)}</span>
          </div>
          <div className="flex justify-between text-[10px]">
            <span className="text-slate-600">Dibayar</span>
            <span className="font-mono">{formatCurrency(invoice.paid_amount)}</span>
          </div>
          <div className="flex justify-between text-[10px] font-bold">
            <span>Sisa tagihan</span>
            <span className="font-mono">{formatCurrency(invoice.remaining_amount)}</span>
          </div>
        </div>
      </section>

      {postedPayments.length > 0 && (
        <section className="relative mt-5 break-inside-avoid">
          <p className="mb-1 text-[9px] font-bold uppercase tracking-[0.14em] text-slate-500">Riwayat Pembayaran</p>
          <table className="w-full border-collapse text-left">
            <thead>
              <tr className="border-y border-slate-300 bg-slate-50 text-[9px] uppercase tracking-wide text-slate-600">
                <th className="px-2 py-1.5 font-bold">No. Pembayaran</th>
                <th className="px-2 py-1.5 font-bold">Metode</th>
                <th className="px-2 py-1.5 font-bold">Waktu</th>
                <th className="px-2 py-1.5 text-right font-bold">Nominal</th>
              </tr>
            </thead>
            <tbody>
              {postedPayments.map((payment) => (
                <tr key={payment.id} className="border-b border-slate-200">
                  <td className="px-2 py-1.5 font-mono">{payment.payment_no}</td>
                  <td className="px-2 py-1.5">{paymentMethodLabel(payment.method)}</td>
                  <td className="px-2 py-1.5 text-[10px] text-slate-600">{formatDateTime(payment.paid_at)}</td>
                  <td className="px-2 py-1.5 text-right font-mono">{formatCurrency(payment.amount)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
      )}

      {(invoice.notes || store.invoice_terms) && (
        <section className="relative mt-5 grid gap-4 text-[10px] text-slate-600">
          {invoice.notes && (
            <div>
              <p className="font-bold uppercase tracking-wide text-slate-500">Catatan</p>
              <p className="whitespace-pre-line">{invoice.notes}</p>
            </div>
          )}
          {store.invoice_terms && (
            <div>
              <p className="font-bold uppercase tracking-wide text-slate-500">Syarat & Ketentuan</p>
              <p className="whitespace-pre-line">{store.invoice_terms}</p>
            </div>
          )}
        </section>
      )}

      <footer className="relative mt-6 border-t border-slate-300 pt-3 text-center text-[9px] text-slate-500">
        <p>{store.invoice_footer || 'Terima kasih atas kepercayaan Anda menggunakan layanan kami.'}</p>
        <p className="mt-1 font-mono">
          {invoice.invoice_no} · Dicetak {formatDateTime(new Date().toISOString())}
        </p>
      </footer>
    </div>
  );
}
