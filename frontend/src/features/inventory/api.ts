import { api } from '@/lib/api-client';
import type { ApiEnvelope } from '@/types/api';
import type {
  AdjustmentPayload,
  CreateSparepartPayload,
  CreateSupplierPayload,
  Movement,
  MovementFilter,
  PurchasePayload,
  PurchaseResult,
  Sparepart,
  SparepartCategory,
  SparepartFilter,
  StockAlert,
  Supplier,
  TopUsed,
  UpdateSparepartPayload,
  Valuation,
} from './types';

export async function listSpareparts(filter: SparepartFilter = {}): Promise<ApiEnvelope<Sparepart[]>> {
  const params = new URLSearchParams();
  if (filter.q) params.set('q', filter.q);
  if (filter.category_id) params.set('category_id', filter.category_id);
  if (filter.supplier_id) params.set('supplier_id', filter.supplier_id);
  if (filter.low_stock) params.set('low_stock', 'true');
  if (filter.is_active !== undefined) params.set('is_active', String(filter.is_active));
  if (filter.page) params.set('page', String(filter.page));
  if (filter.per_page) params.set('per_page', String(filter.per_page));
  const query = params.toString();
  return api.getEnvelope<Sparepart[]>(`/spareparts${query ? `?${query}` : ''}`);
}

export async function getSparepart(id: string): Promise<Sparepart> {
  return api.get<Sparepart>(`/spareparts/${id}`);
}

export async function getSparepartByBarcode(code: string): Promise<Sparepart> {
  return api.get<Sparepart>(`/spareparts/barcode/${encodeURIComponent(code)}`);
}

export async function createSparepart(payload: CreateSparepartPayload): Promise<Sparepart> {
  return api.post<Sparepart>('/spareparts', payload);
}

export async function updateSparepart(id: string, payload: UpdateSparepartPayload): Promise<Sparepart> {
  return api.patch<Sparepart>(`/spareparts/${id}`, payload);
}

export async function deleteSparepart(id: string): Promise<{ message: string }> {
  return api.delete<{ message: string }>(`/spareparts/${id}`);
}

export async function listCategories(): Promise<SparepartCategory[]> {
  return api.get<SparepartCategory[]>('/sparepart-categories');
}

export async function createCategory(name: string): Promise<SparepartCategory> {
  return api.post<SparepartCategory>('/sparepart-categories', { name });
}

export async function updateCategory(id: string, name: string): Promise<SparepartCategory> {
  return api.patch<SparepartCategory>(`/sparepart-categories/${id}`, { name });
}

export async function listSuppliers(q?: string): Promise<Supplier[]> {
  const query = q?.trim() ? `?q=${encodeURIComponent(q.trim())}` : '';
  return api.get<Supplier[]>(`/suppliers${query}`);
}

export async function createSupplier(payload: CreateSupplierPayload): Promise<Supplier> {
  return api.post<Supplier>('/suppliers', payload);
}

export async function updateSupplier(id: string, payload: Partial<CreateSupplierPayload>): Promise<Supplier> {
  return api.patch<Supplier>(`/suppliers/${id}`, payload);
}

export async function listMovements(filter: MovementFilter = {}): Promise<ApiEnvelope<Movement[]>> {
  const params = new URLSearchParams();
  if (filter.sparepart_id) params.set('sparepart_id', filter.sparepart_id);
  if (filter.movement_type) params.set('movement_type', filter.movement_type);
  if (filter.outlet_id) params.set('outlet_id', filter.outlet_id);
  if (filter.from) params.set('from', filter.from);
  if (filter.to) params.set('to', filter.to);
  if (filter.page) params.set('page', String(filter.page));
  if (filter.per_page) params.set('per_page', String(filter.per_page));
  const query = params.toString();
  return api.getEnvelope<Movement[]>(`/inventory/movements${query ? `?${query}` : ''}`);
}

export async function purchaseStock(payload: PurchasePayload): Promise<PurchaseResult> {
  return api.post<PurchaseResult>('/inventory/purchases', payload);
}

export async function adjustStock(payload: AdjustmentPayload): Promise<Movement> {
  return api.post<Movement>('/inventory/adjustments', payload);
}

export async function listStockAlerts(): Promise<StockAlert[]> {
  return api.get<StockAlert[]>('/inventory/alerts');
}

export async function getValuation(): Promise<Valuation> {
  return api.get<Valuation>('/inventory/valuation');
}

export async function listTopUsed(from?: string, to?: string): Promise<TopUsed[]> {
  const params = new URLSearchParams();
  if (from) params.set('from', from);
  if (to) params.set('to', to);
  const query = params.toString();
  return api.get<TopUsed[]>(`/inventory/top-used${query ? `?${query}` : ''}`);
}
