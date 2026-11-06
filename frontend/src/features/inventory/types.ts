export interface Sparepart {
  id: string;
  sku: string;
  barcode?: string | null;
  name: string;
  category_id?: string | null;
  category_name?: string | null;
  supplier_id?: string | null;
  supplier_name?: string | null;
  compatible_models?: string | null;
  stock: number;
  min_stock: number;
  cost_price: number;
  sell_price: number;
  stock_value: number;
  location?: string | null;
  is_active: boolean;
  low_stock: boolean;
  last_stock_at?: string | null;
  created_at: string;
  updated_at: string;
}

export interface SparepartFilter {
  q?: string;
  category_id?: string;
  supplier_id?: string;
  low_stock?: boolean;
  is_active?: boolean;
  page?: number;
  per_page?: number;
}

export interface CreateSparepartPayload {
  sku: string;
  barcode?: string;
  name: string;
  category_id?: string;
  supplier_id?: string;
  compatible_models?: string;
  stock?: number;
  min_stock?: number;
  cost_price?: number;
  sell_price?: number;
  location?: string;
  is_active?: boolean;
}

export type UpdateSparepartPayload = Partial<CreateSparepartPayload>;

export interface SparepartCategory {
  id: string;
  name: string;
  sparepart_count: number;
  created_at: string;
}

export interface Supplier {
  id: string;
  name: string;
  contact_name?: string | null;
  phone?: string | null;
  address?: string | null;
  notes?: string | null;
  created_at: string;
  updated_at: string;
}

export interface CreateSupplierPayload {
  name: string;
  contact_name?: string;
  phone?: string;
  address?: string;
  notes?: string;
}

export interface Movement {
  id: string;
  sparepart_id: string;
  sparepart_name: string;
  sku: string;
  outlet_id: string;
  outlet_name: string;
  movement_type: string;
  qty: number;
  stock_before: number;
  stock_after: number;
  unit_cost: number;
  reference_type: string;
  reference_id?: string | null;
  notes?: string | null;
  created_by?: string | null;
  created_at: string;
}

export interface MovementFilter {
  sparepart_id?: string;
  movement_type?: string;
  outlet_id?: string;
  from?: string;
  to?: string;
  page?: number;
  per_page?: number;
}

export interface PurchaseItemPayload {
  sparepart_id: string;
  qty: number;
  unit_cost: number;
}

export interface PurchasePayload {
  supplier_id?: string;
  reference_no?: string;
  purchased_at?: string;
  items: PurchaseItemPayload[];
  notes?: string;
}

export interface PurchaseResult {
  reference_id: string;
  purchased_at: string;
  total_cost: number;
  items: Movement[];
}

export interface AdjustmentPayload {
  sparepart_id: string;
  new_stock: number;
  reason: string;
  notes?: string;
}

export interface StockAlert {
  id: string;
  sku: string;
  name: string;
  category_name?: string | null;
  stock: number;
  min_stock: number;
  shortage: number;
}

export interface Valuation {
  total_items: number;
  total_stock: number;
  total_value: number;
}

export interface TopUsed {
  sparepart_id: string;
  sku: string;
  name: string;
  used_qty: number;
  movement_count: number;
}
