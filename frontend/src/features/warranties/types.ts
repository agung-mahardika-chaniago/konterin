export type WarrantyStatus = 'active' | 'expired' | 'void';
export type WarrantyType = 'service' | 'part' | 'both';
export type ClaimStatus = 'submitted' | 'approved' | 'rejected' | 'in_progress' | 'completed';

export interface CustomerRef {
  id: string;
  name: string;
  phone: string;
}

export interface DeviceRef {
  brand: string;
  model: string;
}

export interface ServiceRef {
  id: string;
  ticket_no: string;
  status: string;
}

export interface OutletRef {
  id: string;
  name: string;
}

export interface InvoiceRef {
  id: string;
  invoice_no: string;
  status: string;
  total_amount: number;
  paid_amount: number;
  remaining_amount: number;
}

export interface Warranty {
  id: string;
  warranty_no: string;
  type: WarrantyType;
  status: WarrantyStatus;
  start_date: string;
  end_date: string;
  days: number;
  terms?: string | null;
  customer: CustomerRef;
  device: DeviceRef;
  service_order: ServiceRef;
  outlet: OutletRef;
  claim_count: number;
  invoice?: InvoiceRef | null;
  claims?: Claim[];
  created_at: string;
}

export interface Claim {
  id: string;
  claim_no: string;
  warranty_id: string;
  warranty_no: string;
  status: ClaimStatus;
  complaint: string;
  resolution?: string | null;
  customer: CustomerRef;
  device: DeviceRef;
  service_order: ServiceRef;
  outlet: OutletRef;
  handled_by?: string | null;
  rework_service_order?: ServiceRef;
  claimed_at: string;
  resolved_at?: string | null;
  created_at: string;
  updated_at: string;
}

export interface WarrantyFilter {
  status?: string;
  q?: string;
  outlet_id?: string;
  expiring_in_days?: number;
  page?: number;
  per_page?: number;
}

export interface ClaimFilter {
  status?: string;
  q?: string;
  outlet_id?: string;
  page?: number;
  per_page?: number;
}

export interface CreateClaimRequest {
  complaint: string;
}

export interface UpdateClaimRequest {
  status?: string;
  resolution?: string;
}

export interface ReworkResult {
  service_order_id: string;
  ticket_no: string;
  status: string;
  claim_id: string;
  claim_status: string;
  invoice_no: string;
  created_at: string;
}
