import * as React from 'react';
import { Printer, ReceiptText, FileText } from 'lucide-react';

import { Alert } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Skeleton } from '@/components/ui/skeleton';
import { getInvoicePrint } from '../invoices/api';
import type { InvoicePrintPayload } from '../invoices/types';
import { InvoiceA4 } from './invoice-a4';
import { ReceiptThermal } from './receipt-thermal';

export type PrintTemplate = 'a4' | 'receipt';

interface PrintInvoiceModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  invoiceId: string | null;
  initialTemplate?: PrintTemplate;
}

export function PrintInvoiceModal({ open, onOpenChange, invoiceId, initialTemplate = 'a4' }: PrintInvoiceModalProps) {
  const [payload, setPayload] = React.useState<InvoicePrintPayload | null>(null);
  const [template, setTemplate] = React.useState<PrintTemplate>(initialTemplate);
  const [width, setWidth] = React.useState<'58' | '80'>('58');
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  React.useEffect(() => {
    if (!open) return;
    setTemplate(initialTemplate);
    setWidth('58');
  }, [open, initialTemplate]);

  React.useEffect(() => {
    if (!open || !invoiceId) return;
    let active = true;
    setLoading(true);
    setError(null);
    setPayload(null);
    getInvoicePrint(invoiceId)
      .then((data) => {
        if (!active) return;
        setPayload(data);
        setWidth(data.invoice.store.receipt_paper_width === 80 ? '80' : '58');
      })
      .catch((err: unknown) => {
        if (!active) return;
        setError(err instanceof Error ? err.message : 'Gagal memuat data cetak invoice.');
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [open, invoiceId]);

  const handlePrint = () => {
    if (!payload) return;
    const content = document.getElementById('printable-invoice-content')?.innerHTML;
    if (!content) {
      setError('Pratinjau cetak belum siap. Coba muat ulang dialog.');
      return;
    }

    const printWindow = window.open('', '_blank', template === 'a4' ? 'width=900,height=760' : 'width=450,height=650');
    if (!printWindow) {
      window.print();
      return;
    }

    const styles = Array.from(document.querySelectorAll('style, link[rel="stylesheet"]'))
      .map((element) => element.outerHTML)
      .join('\n');
    const pageSize = template === 'a4' ? 'A4' : `${width}mm auto`;
    const bodyWidth = template === 'a4' ? '210mm' : `${width}mm`;

    printWindow.document.write(`
      <!DOCTYPE html>
      <html>
        <head>
          <title>${template === 'a4' ? 'Invoice' : 'Struk'} - ${payload.invoice.invoice_no}</title>
          ${styles}
          <style>
            @page {
              size: ${pageSize};
              margin: 0;
            }
            html, body {
              margin: 0;
              padding: 0;
              background: #ffffff;
              -webkit-print-color-adjust: exact;
              print-color-adjust: exact;
            }
            body {
              width: ${bodyWidth};
              ${template === 'a4' ? 'margin: 0 auto;' : ''}
            }
            .no-print { display: none !important; }
          </style>
        </head>
        <body>
          ${content}
          <script>
            window.onload = function() {
              window.focus();
              window.print();
              setTimeout(function() { window.close(); }, 600);
            };
          </script>
        </body>
      </html>
    `);
    printWindow.document.close();
  };

  const invoice = payload?.invoice;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-5xl" onClose={() => onOpenChange(false)}>
        <DialogHeader>
          <DialogTitle>Cetak Invoice & Struk</DialogTitle>
          <DialogDescription>
            {invoice
              ? `${invoice.invoice_no} · ${invoice.customer_name} · ${invoice.outlet_name}`
              : 'Pilih format kertas, periksa pratinjau, lalu cetak.'}
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-wrap items-center justify-center gap-2 border-y border-border py-3">
          <button
            type="button"
            onClick={() => setTemplate('a4')}
            className={`inline-flex items-center gap-1.5 rounded-control border px-3 py-1.5 text-xs font-medium transition-colors ${
              template === 'a4'
                ? 'border-brand-900 bg-brand-900 text-white'
                : 'border-border bg-surface text-muted hover:text-ink'
            }`}
          >
            <FileText className="h-3.5 w-3.5" aria-hidden="true" />
            Invoice A4
          </button>
          <button
            type="button"
            onClick={() => setTemplate('receipt')}
            className={`inline-flex items-center gap-1.5 rounded-control border px-3 py-1.5 text-xs font-medium transition-colors ${
              template === 'receipt'
                ? 'border-brand-900 bg-brand-900 text-white'
                : 'border-border bg-surface text-muted hover:text-ink'
            }`}
          >
            <ReceiptText className="h-3.5 w-3.5" aria-hidden="true" />
            Struk Thermal
          </button>
          {template === 'receipt' && (
            <div className="ml-2 inline-flex rounded-control border border-border bg-background p-0.5">
              <button
                type="button"
                onClick={() => setWidth('58')}
                className={`rounded px-2.5 py-1 text-[11px] font-medium transition-colors ${
                  width === '58' ? 'bg-surface text-ink shadow-sm' : 'text-muted hover:text-ink'
                }`}
              >
                58 mm
              </button>
              <button
                type="button"
                onClick={() => setWidth('80')}
                className={`rounded px-2.5 py-1 text-[11px] font-medium transition-colors ${
                  width === '80' ? 'bg-surface text-ink shadow-sm' : 'text-muted hover:text-ink'
                }`}
              >
                80 mm
              </button>
            </div>
          )}
        </div>

        {error && (
          <div className="pt-4">
            <Alert variant="error">{error}</Alert>
          </div>
        )}

        {template === 'a4' && invoice?.status === 'draft' && (
          <div className="pt-4">
            <Alert variant="info">
              Invoice masih draft. Dokumen akan tercetak dengan tanda DRAFT dan item snapshot belum tersedia.
            </Alert>
          </div>
        )}

        <div className="mt-4 max-h-[58vh] overflow-auto rounded-lg border border-slate-200 bg-slate-100 p-4">
          {loading ? (
            <div className="mx-auto w-[210mm] max-w-full space-y-4 bg-white p-8">
              <Skeleton className="h-10 w-56" />
              <Skeleton className="h-4 w-80" />
              <Skeleton className="h-28 w-full" />
              <Skeleton className="h-48 w-full" />
            </div>
          ) : invoice ? (
            <div id="printable-invoice-content" className="mx-auto w-max bg-white shadow-md">
              {template === 'a4' ? (
                <InvoiceA4 invoice={invoice} />
              ) : (
                <ReceiptThermal invoice={invoice} width={width} />
              )}
            </div>
          ) : (
            <div className="py-12 text-center text-sm text-muted">Data invoice tidak tersedia.</div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Tutup
          </Button>
          <Button onClick={handlePrint} disabled={!invoice || loading} className="gap-2">
            <Printer className="h-4 w-4" aria-hidden="true" />
            {template === 'a4' ? 'Cetak Invoice' : 'Cetak Struk'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
