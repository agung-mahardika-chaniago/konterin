export interface CustomerRef {
  id: string;
  name: string;
  phone: string;
}

export interface DeviceRef {
  id: string;
  brand: string;
  model: string;
  imei?: string | null;
  color?: string | null;
}

export interface TechnicianRef {
  id: string;
  name: string;
}

export interface ServiceOrderListItem {
  id: string;
  ticket_no: string;
  status: string;
  priority: 'normal' | 'express';
  customer: CustomerRef;
  device: DeviceRef;
  assigned_technician?: TechnicianRef | null;
  complaint: string;
  estimated_cost: number;
  estimated_done_at?: string | null;
  received_at: string;
  completed_at?: string | null;
  picked_up_at?: string | null;
  credential_configured: boolean;
  photo_count: number;
}

export interface ServicePhoto {
  id: string;
  stage: 'intake' | 'before' | 'after' | 'testing' | 'other' | string;
  file_path: string;
  mime_type: string;
  size_bytes: number;
  is_public: boolean;
  created_at: string;
}

export interface StatusHistoryEntry {
  id: string;
  from_status?: string | null;
  to_status: string;
  notes?: string | null;
  changed_by_id?: string | null;
  changed_by?: string | null;
  created_at: string;
}

export interface InvoiceRef {
  id: string;
  invoice_no: string;
  status: string;
  total_amount: number;
  paid_amount: number;
  remaining_amount: number;
}

export interface StoreRef {
  name: string;
  address?: string | null;
  phone?: string | null;
}

export interface ServiceOrderDetail extends ServiceOrderListItem {
  completeness?: string | null;
  physical_condition?: string | null;
  notes?: string | null;
  internal_notes?: string | null;
  credential_type: string;
  photos: ServicePhoto[];
  timeline: StatusHistoryEntry[];
  invoice?: InvoiceRef | null;
  diagnostic?: Diagnostic | null;
  items?: ServiceOrderItem[];
  store: StoreRef;
  public_token?: string | null;
  tracking_url?: string | null;
  created_at: string;
  updated_at: string;
}

export interface Diagnostic {
  id: string;
  diagnosis: string | null;
  findings: string | null;
  action_taken: string | null;
  test_result: string | null;
  technician_notes: string | null;
  diagnosed_by_id?: string | null;
  diagnosed_by?: string | null;
  diagnosed_at: string | null;
  estimated_done_at: string | null;
  updated_at: string;
}

export interface ServiceOrderItem {
  id: string;
  item_type: 'service' | 'sparepart' | 'custom' | string;
  sparepart_id?: string | null;
  description: string;
  qty: number;
  unit_price: number;
  cost_price: number;
  discount_amount: number;
  subtotal: number;
  warranty_days: number;
  added_by?: string | null;
  created_at: string;
}

export interface SparepartOption {
  id: string;
  sku: string;
  name: string;
  stock: number;
  cost_price: number;
  sell_price: number;
  location: string;
}

export interface ChangeStatusRequest {
  status: string;
  notes?: string | null;
  cancel_reason?: string | null;
  override_reason?: string | null;
}

export interface AssignTechnicianRequest {
  technician_id?: string | null;
  priority?: 'normal' | 'express';
}

export interface UpsertDiagnosticRequest {
  diagnosis?: string | null;
  findings?: string | null;
  action_taken?: string | null;
  test_result?: string | null;
  technician_notes?: string | null;
  estimated_done_at?: string | null;
}

export interface AddServiceItemRequest {
  item_type: 'service' | 'sparepart' | 'custom';
  sparepart_id?: string | null;
  description?: string;
  qty: number;
  unit_price?: number | null;
  discount_amount?: number;
  warranty_days?: number;
}

export interface UpdateServiceItemRequest {
  description?: string;
  qty?: number;
  unit_price?: number;
  discount_amount?: number;
  warranty_days?: number;
}

export interface ServiceOrderCreated {
  id: string;
  ticket_no: string;
  public_token: string;
  tracking_url: string;
  status: string;
  customer_id: string;
  device_id: string;
  invoice_no: string;
  created_at: string;
}

export interface TechnicianOption {
  id: string;
  name: string;
}

export interface ServiceFilter {
  q?: string;
  status?: string;
  technician_id?: string;
  outlet_id?: string;
  from?: string;
  to?: string;
  customer_id?: string;
  page?: number;
  per_page?: number;
}

export interface CreateServiceRequest {
  customer: {
    id?: string | null;
    name: string;
    phone: string;
    address?: string | null;
  };
  device: {
    brand: string;
    model: string;
    imei?: string | null;
    color?: string | null;
  };
  complaint: string;
  completeness?: string | null;
  physical_condition?: string | null;
  credential?: {
    type: 'pin' | 'password' | 'pattern' | 'none' | string;
    value: string;
  } | null;
  estimated_cost?: number;
  estimated_done_at?: string | null;
  assigned_technician_id?: string | null;
  priority?: 'normal' | 'express';
  notes?: string | null;
  internal_notes?: string | null;
}

export interface UpdateServiceRequest {
  complaint?: string;
  completeness?: string;
  physical_condition?: string;
  estimated_cost?: number;
  estimated_done_at?: string | null;
  assigned_technician_id?: string | null;
  priority?: 'normal' | 'express';
  notes?: string;
  internal_notes?: string;
}
