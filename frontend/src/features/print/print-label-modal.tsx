import * as React from 'react';
import { Printer } from 'lucide-react';

import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { ThermalLabel } from './thermal-label';
import type { ServiceOrderDetail, ServiceOrderListItem } from '../services/types';

interface PrintLabelModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  order: ServiceOrderDetail | ServiceOrderListItem | null;
}

export function PrintLabelModal({ open, onOpenChange, order }: PrintLabelModalProps) {
  const [width, setWidth] = React.useState<'58' | '80'>('58');

  if (!order) return null;

  const handlePrint = () => {
    const printWindow = window.open('', '_blank', 'width=450,height=600');
    if (!printWindow) {
      window.print();
      return;
    }

    const labelHtml = document.getElementById('printable-label-content')?.innerHTML ?? '';
    const styles = Array.from(document.querySelectorAll('style, link[rel="stylesheet"]'))
      .map((el) => el.outerHTML)
      .join('\n');

    printWindow.document.write(`
      <!DOCTYPE html>
      <html>
        <head>
          <title>Label Servis - ${order.ticket_no}</title>
          ${styles}
          <style>
            @page {
              size: ${width}mm auto;
              margin: 0;
            }
            body {
              margin: 0;
              padding: 0;
              background: white;
              font-family: monospace;
              -webkit-print-color-adjust: exact;
              print-color-adjust: exact;
            }
            .no-print { display: none !important; }
          </style>
        </head>
        <body>
          <div style="width: ${width}mm; padding: 4px; box-sizing: border-box;">
            ${labelHtml}
          </div>
          <script>
            window.onload = function() {
              window.focus();
              window.print();
              setTimeout(function() { window.close(); }, 500);
            };
          </script>
        </body>
      </html>
    `);
    printWindow.document.close();
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Cetak Label Stiker Servis</DialogTitle>
          <DialogDescription>
            Pilih ukuran kertas thermal printer dan pratinjau label sebelum mencetak.
          </DialogDescription>
        </DialogHeader>

        <div className="flex items-center justify-center gap-3 py-2 border-y border-border">
          <span className="text-xs font-semibold text-muted">Ukuran kertas:</span>
          <div className="inline-flex rounded-lg border border-border bg-background p-0.5">
            <button
              type="button"
              onClick={() => setWidth('58')}
              className={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${
                width === '58' ? 'bg-surface text-ink shadow-sm' : 'text-muted hover:text-ink'
              }`}
            >
              Thermal 58 mm
            </button>
            <button
              type="button"
              onClick={() => setWidth('80')}
              className={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${
                width === '80' ? 'bg-surface text-ink shadow-sm' : 'text-muted hover:text-ink'
              }`}
            >
              Thermal 80 mm
            </button>
          </div>
        </div>

        {/* Live Preview Container */}
        <div className="py-4 bg-slate-50 flex items-center justify-center rounded-lg border border-slate-200 overflow-x-auto min-h-[300px]">
          <div id="printable-label-content" className="bg-white shadow-md p-1 rounded">
            <ThermalLabel order={order} width={width} />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Tutup
          </Button>
          <Button onClick={handlePrint} className="gap-2">
            <Printer className="h-4 w-4" aria-hidden="true" />
            Cetak Label
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
