import { cn } from '@/lib/utils';
import { formatCurrency } from '@/features/dashboard/format';
import { invoiceStatusLabel, itemTypeLabel, paymentMethodLabel } from '../invoices/format';
import type { InvoiceDetail } from '../invoices/types';

interface ReceiptThermalProps {
  invoice: InvoiceDetail;
  width?: '58' | '80';
  className?: string;
}

function formatQty(value: number): string {
  return Number.isInteger(value) ? String(value) : value.toLocaleString('id-ID', { maximumFractionDigits: 2 });
}

export function ReceiptThermal({ invoice, width = '58', className }: ReceiptThermalProps) {
  const store = invoice.store;
  const is58 = width === '58';
  const postedPayments = invoice.payments.filter((payment) => payment.status === 'posted');
  const lastPayment = postedPayments[postedPayments.length - 1];

  const paidAt = lastPayment ? new Date(lastPayment.paid_at) : new Date(invoice.issued_at ?? invoice.created_at);
  const paidAtLabel = paidAt.toLocaleString('id-ID', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });

  return (
    <div
      className={cn(
        'bg-white font-mono text-black',
        is58 ? 'w-[58mm] max-w-[58mm] text-[9px] leading-snug' : 'w-[80mm] max-w-[80mm] text-[10px] leading-normal',
        className,
      )}
    >
      <div className="border-b border-dashed border-black pb-2 text-center">
        <p className={cn('font-bold uppercase tracking-wide', is58 ? 'text-[11px]' : 'text-[12px]')}>{store.name}</p>
        {store.legal_name && store.legal_name !== store.name && (
          <p className="text-[8px] text-slate-600">{store.legal_name}</p>
        )}
        {store.address && <p className="whitespace-pre-line text-[8px] text-slate-600">{store.address}</p>}
        <p className="text-[8px] text-slate-600">
          {[store.phone, store.email].filter(Boolean).join(' · ')}
        </p>
      </div>

      <div className="border-b border-dashed border-black py-2">
        <p className="text-center text-[10px] font-bold uppercase tracking-[0.14em]">Struk Pembayaran</p>
        <div className="mt-1 flex justify-between gap-2">
          <span className="text-slate-600">No. Invoice</span>
          <span className="text-right font-bold">{invoice.invoice_no}</span>
        </div>
        <div className="flex justify-between gap-2">
          <span className="text-slate-600">Tiket</span>
          <span className="text-right">{invoice.ticket_no}</span>
        </div>
        <div className="flex justify-between gap-2">
          <span className="text-slate-600">Tanggal</span>
          <span className="text-right">{paidAtLabel}</span>
        </div>
        <div className="flex justify-between gap-2">
          <span className="text-slate-600">Pelanggan</span>
          <span className="max-w-[65%] truncate text-right">{invoice.customer_name}</span>
        </div>
        {lastPayment?.received_by && (
          <div className="flex justify-between gap-2">
            <span className="text-slate-600">Kasir</span>
            <span className="max-w-[65%] truncate text-right">{lastPayment.received_by}</span>
          </div>
        )}
        <div className="flex justify-between gap-2">
          <span className="text-slate-600">Status</span>
          <span className="text-right font-bold uppercase">{invoiceStatusLabel(invoice.status)}</span>
        </div>
      </div>

      <div className="border-b border-dashed border-black py-2">
        <div className="flex justify-between gap-2 text-[8px] font-bold uppercase tracking-wide text-slate-600">
          <span>Item</span>
          <span>Subtotal</span>
        </div>
        {invoice.items.length === 0 ? (
          <p className="py-2 text-center text-[8px] italic text-slate-500">Belum ada item invoice.</p>
        ) : (
          invoice.items.map((item) => (
            <div key={item.id} className="mt-1.5 break-inside-avoid">
              <p className="break-words font-medium">{item.description}</p>
              <div className="flex justify-between gap-2 text-[8px] text-slate-600">
                <span>
                  {formatQty(item.qty)} x {formatCurrency(item.unit_price)}
                  {item.discount_amount > 0 ? ` (disc ${formatCurrency(item.discount_amount)})` : ''}
                </span>
                <span className="shrink-0 font-semibold text-black">{formatCurrency(item.subtotal)}</span>
              </div>
              <p className="text-[7px] uppercase tracking-wide text-slate-500">
                {itemTypeLabel(item.item_type)}
                {item.warranty_days > 0 ? ` · garansi ${item.warranty_days} hari` : ''}
              </p>
            </div>
          ))
        )}
      </div>

      <div className="border-b border-dashed border-black py-2">
        <div className="flex justify-between gap-2">
          <span className="text-slate-600">Subtotal jasa</span>
          <span>{formatCurrency(invoice.subtotal_service)}</span>
        </div>
        <div className="flex justify-between gap-2">
          <span className="text-slate-600">Subtotal sparepart</span>
          <span>{formatCurrency(invoice.subtotal_part)}</span>
        </div>
        {invoice.discount_amount > 0 && (
          <div className="flex justify-between gap-2">
            <span className="text-slate-600">Diskon</span>
            <span>- {formatCurrency(invoice.discount_amount)}</span>
          </div>
        )}
        <div className="mt-1 flex justify-between gap-2 border-t border-black pt-1 text-[11px] font-bold">
          <span>TOTAL</span>
          <span>{formatCurrency(invoice.total_amount)}</span>
        </div>
        <div className="flex justify-between gap-2">
          <span className="text-slate-600">Dibayar</span>
          <span>{formatCurrency(invoice.paid_amount)}</span>
        </div>
        <div className="flex justify-between gap-2 font-bold">
          <span>Sisa</span>
          <span>{formatCurrency(invoice.remaining_amount)}</span>
        </div>
      </div>

      {postedPayments.length > 0 && (
        <div className="border-b border-dashed border-black py-2">
          <p className="text-[8px] font-bold uppercase tracking-wide text-slate-600">Pembayaran</p>
          {postedPayments.map((payment) => (
            <div key={payment.id} className="mt-1">
              <div className="flex justify-between gap-2">
                <span>{paymentMethodLabel(payment.method)}</span>
                <span className="font-semibold">{formatCurrency(payment.amount)}</span>
              </div>
              <div className="flex justify-between gap-2 text-[7px] text-slate-500">
                <span>{payment.payment_no}</span>
                <span>
                  {new Date(payment.paid_at).toLocaleDateString('id-ID', {
                    day: '2-digit',
                    month: '2-digit',
                    year: '2-digit',
                  })}
                </span>
              </div>
              {payment.reference_no && <p className="text-[7px] text-slate-500">Ref: {payment.reference_no}</p>}
            </div>
          ))}
        </div>
      )}

      <div className="pt-2 text-center text-[8px] text-slate-700">
        <p className="text-[10px] font-bold uppercase tracking-wide">
          {invoice.remaining_amount <= 0 ? 'Lunas' : 'Belum lunas'}
        </p>
        <p className="mt-1">{store.invoice_footer || 'Terima kasih atas kunjungan Anda.'}</p>
        <p className="mt-0.5 text-[7px] text-slate-500">
          Barang yang sudah diambil dianggap telah diperiksa dan diterima dengan baik.
        </p>
        <p className="mt-1 font-mono text-[7px] text-slate-500">{invoice.invoice_no}</p>
      </div>
    </div>
  );
}
