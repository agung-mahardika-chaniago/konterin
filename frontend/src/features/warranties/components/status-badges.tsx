import { Badge } from '@/components/ui/badge';
import { CLAIM_STATUS, WARRANTY_STATUS, WARRANTY_TYPE } from '../format';
import type { ClaimStatus, WarrantyStatus, WarrantyType } from '../types';

export function WarrantyStatusBadge({ status }: { status: string }) {
  const meta = WARRANTY_STATUS[status as WarrantyStatus];
  return <Badge variant={meta?.variant ?? 'neutral'}>{meta?.label ?? status}</Badge>;
}

export function WarrantyTypeBadge({ type }: { type: string }) {
  const meta = WARRANTY_TYPE[type as WarrantyType];
  return <Badge variant={meta?.variant ?? 'neutral'}>{meta?.label ?? type}</Badge>;
}

export function ClaimStatusBadge({ status }: { status: string }) {
  const meta = CLAIM_STATUS[status as ClaimStatus];
  return <Badge variant={meta?.variant ?? 'neutral'}>{meta?.label ?? status}</Badge>;
}
