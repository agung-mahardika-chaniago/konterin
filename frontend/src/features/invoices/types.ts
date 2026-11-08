export type InvoiceStatus = 'draft' | 'unpaid' | 'partial' | 'paid' | 'void';
export type PaymentMethod = 'cash' | 'bank_transfer' | 'qris' | 'card' | 'other';
export type PaymentStatus = 'posted' | 'void';

export interface InvoiceStore {
  name: string;
  legal_name?: string | null;
  logo_path?: string | null;
  address?: string | null;
  phone?: string | null;
  email?: string | null;
  tax_id?: string | null;
  invoice_terms?: string | null;
  invoice_footer?: string | null;
  timezone: string;
  receipt_paper_width: number;
  currency: string;
}

export interface InvoiceServiceOrder {
  id: string;
  ticket_no: string;
  status: string;
  priority: string;
  device_brand: string;
  device_model: string;
  device_imei?: string | null;
  device_color?: string | null;
  complaint: string;
  received_at: string;
  completed_at?: string | null;
  picked_up_at?: string | null;
  technician_name?: string | null;
  public_tracking_url?: string | null;
}

export interface InvoiceItem {
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
}

export interface InvoicePayment {
  id: string;
  payment_no: string;
  invoice_id: string;
  invoice_no: string;
  ticket_no: string;
  customer_name: string;
  method: PaymentMethod;
  amount: number;
  paid_at: string;
  reference_no?: string | null;
  notes?: string | null;
  status: PaymentStatus;
  received_by?: string | null;
  voided_by?: string | null;
  voided_at?: string | null;
  void_reason?: string | null;
  created_at: string;
}

export interface InvoiceListItem {
  id: string;
  invoice_no: string;
  status: InvoiceStatus;
  service_order_id: string;
  ticket_no: string;
  service_status: string;
  customer_id: string;
  customer_name: string;
  customer_phone: string;
  outlet_id: string;
  outlet_name: string;
  subtotal_service: number;
  subtotal_part: number;
  discount_type: 'amount' | 'percent';
  discount_value: number;
  discount_amount: number;
  tax_amount: number;
  total_amount: number;
  paid_amount: number;
  remaining_amount: number;
  issued_at?: string | null;
  due_at?: string | null;
  created_at: string;
  updated_at: string;
}

export interface InvoiceDetail extends InvoiceListItem {
  notes?: string | null;
  void_reason?: string | null;
  voided_by?: string | null;
  voided_at?: string | null;
  service_order: InvoiceServiceOrder;
  items: InvoiceItem[];
  payments: InvoicePayment[];
  store: InvoiceStore;
}

export interface InvoicePrintPayload {
  template: string;
  paper: string;
  invoice: InvoiceDetail;
}

export interface InvoiceFilter {
  q?: string;
  status?: string;
  outlet_id?: string;
  from?: string;
  to?: string;
  page?: number;
  per_page?: number;
}

export interface UpdateInvoiceRequest {
  discount_type?: 'amount' | 'percent';
  discount_value?: number;
  notes?: string | null;
  due_at?: string | null;
}

export interface CreatePaymentRequest {
  method: PaymentMethod;
  amount: number;
  paid_at?: string | null;
  reference_no?: string | null;
  notes?: string | null;
}

export interface PaymentMutation {
  payment: InvoicePayment;
  invoice: {
    id: string;
    invoice_no: string;
    status: InvoiceStatus;
    total_amount: number;
    paid_amount: number;
    remaining_amount: number;
  };
}

export interface PaymentFilter {
  q?: string;
  status?: string;
  method?: string;
  outlet_id?: string;
  from?: string;
  to?: string;
  page?: number;
  per_page?: number;
}

export interface PaymentListItem {
  id: string;
  payment_no: string;
  invoice_id: string;
  invoice_no: string;
  ticket_no: string;
  customer_name: string;
  customer_phone: string;
  outlet_id: string;
  outlet_name: string;
  method: PaymentMethod;
  amount: number;
  paid_at: string;
  reference_no?: string | null;
  notes?: string | null;
  status: PaymentStatus;
  received_by?: string | null;
  voided_by?: string | null;
  voided_at?: string | null;
  void_reason?: string | null;
  created_at: string;
}

export interface MethodSummary {
  method: PaymentMethod;
  amount: number;
  count: number;
}

export interface DailySummary {
  date: string;
  outlet_id: string;
  outlet_name: string;
  total_amount: number;
  posted_count: number;
  invoice_count: number;
  voided_amount: number;
  voided_count: number;
  methods: MethodSummary[];
}
