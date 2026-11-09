import * as React from 'react';
import QRCode from 'qrcode';

import { cn } from '@/lib/utils';
import type { ServiceOrderDetail, ServiceOrderListItem } from '../services/types';

interface ThermalLabelProps {
  order: ServiceOrderDetail | ServiceOrderListItem;
  width?: '58' | '80';
  store?: {
    name: string;
    address?: string | null;
    phone?: string | null;
  };
  trackingUrl?: string | null;
  className?: string;
}

export function ThermalLabel({
  order,
  width = '58',
  store,
  trackingUrl,
  className,
}: ThermalLabelProps) {
  const [qrCodeUrl, setQrCodeUrl] = React.useState<string>('');

  const storedTrackingUrl = 'tracking_url' in order ? order.tracking_url : null;
  const qrTarget =
    trackingUrl ||
    storedTrackingUrl ||
    `${window.location.origin}/track?ticket=${encodeURIComponent(order.ticket_no)}`;

  React.useEffect(() => {
    let active = true;
    QRCode.toDataURL(qrTarget, {
      width: width === '58' ? 90 : 120,
      margin: 1,
      errorCorrectionLevel: 'M',
    })
      .then((url) => {
        if (active) setQrCodeUrl(url);
      })
      .catch((err) => console.error('QR code generation failed', err));

    return () => {
      active = false;
    };
  }, [qrTarget, width]);

  const receivedDate = new Date(order.received_at).toLocaleDateString('id-ID', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });

  const storeInfo = ('store' in order && order.store) || store || {
    name: 'FixPhone Service',
    address: null,
    phone: null,
  };

  const is58 = width === '58';

  return (
    <div
      className={cn(
        'bg-white text-black font-mono border border-dashed border-slate-300 p-3 mx-auto select-none print:border-none print:p-1 print:m-0 print:shadow-none print:w-full',
        is58 ? 'w-[58mm] max-w-[58mm] text-[10px] leading-tight' : 'w-[80mm] max-w-[80mm] text-[11px] leading-normal',
        className,
      )}
    >
      {/* Header Toko */}
      <div className="text-center pb-2 border-b border-black">
        <p className="font-bold uppercase tracking-wider text-[11px]">{storeInfo.name}</p>
        {storeInfo.address && <p className="text-[8px] print:text-[8px] text-slate-600 line-clamp-1">{storeInfo.address}</p>}
        {storeInfo.phone && <p className="text-[9px] print:text-[9px] font-semibold">{storeInfo.phone}</p>}
      </div>

      {/* Ticket No & QR */}
      <div className="flex items-center justify-between gap-1 py-2 border-b border-black">
        <div className="flex-1 min-w-0 pr-1">
          <span className="text-[8px] font-bold uppercase tracking-widest text-slate-600">No. Tiket Servis</span>
          <p className="text-sm font-black tracking-tight break-all font-sans">{order.ticket_no}</p>
          <p className="text-[8px] text-slate-700">{receivedDate}</p>
          {order.priority === 'express' && (
            <span className="inline-block mt-0.5 bg-black text-white px-1 py-0.2 text-[8px] font-bold uppercase">
              ⚡ EXPRESS
            </span>
          )}
        </div>
        {qrCodeUrl && (
          <div className="shrink-0 flex flex-col items-center">
            <img src={qrCodeUrl} alt="QR Tiket" className={is58 ? 'w-16 h-16' : 'w-20 h-20'} />
            <span className="text-[7px] text-slate-500">Scan status</span>
          </div>
        )}
      </div>

      {/* Pelanggan & Perangkat */}
      <div className="py-2 space-y-1 text-left border-b border-black">
        <div className="flex justify-between items-start gap-1">
          <span className="font-semibold text-slate-700 shrink-0">Pelanggan:</span>
          <span className="font-bold text-right truncate">
            {order.customer.name} ({order.customer.phone})
          </span>
        </div>
        <div className="flex justify-between items-start gap-1">
          <span className="font-semibold text-slate-700 shrink-0">Unit:</span>
          <span className="font-bold text-right truncate">
            {order.device.brand} {order.device.model}
            {order.device.color ? ` (${order.device.color})` : ''}
          </span>
        </div>
        {order.device.imei && (
          <div className="flex justify-between items-start gap-1 text-[9px]">
            <span className="text-slate-600 shrink-0">IMEI/SN:</span>
            <span className="text-right truncate">{order.device.imei}</span>
          </div>
        )}
        <div className="pt-0.5">
          <span className="font-semibold text-slate-700 block text-[9px]">Keluhan:</span>
          <p className="font-medium text-[9px] line-clamp-2 italic">{order.complaint}</p>
        </div>
      </div>

      {/* Footer */}
      <div className="pt-2 text-center text-[8px] text-slate-700">
        <p className="font-bold">Simpan label ini untuk bukti tanda terima.</p>
        <p className="text-[7px] text-slate-500 mt-0.5">Lacak status servis di link QR di atas.</p>
      </div>
    </div>
  );
}
