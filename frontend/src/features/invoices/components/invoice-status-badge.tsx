import { Badge } from '@/components/ui/badge';
import { INVOICE_STATUS } from '../format';
import type { InvoiceStatus } from '../types';

export function InvoiceStatusBadge({ status, className }: { status: string; className?: string }) {
  const config = INVOICE_STATUS[status as InvoiceStatus];

  return (
    <Badge variant={config?.variant ?? 'neutral'} className={className}>
      {config?.label ?? status}
    </Badge>
  );
}
