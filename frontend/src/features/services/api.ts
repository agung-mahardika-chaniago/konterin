import { api, apiRequest } from '@/lib/api-client';
import type { ApiEnvelope } from '@/types/api';
import type {
  AddServiceItemRequest,
  AssignTechnicianRequest,
  ChangeStatusRequest,
  CreateServiceRequest,
  Diagnostic,
  ServiceFilter,
  ServiceOrderDetail,
  ServiceOrderItem,
  ServiceOrderListItem,
  ServicePhoto,
  SparepartOption,
  StatusHistoryEntry,
  TechnicianOption,
  UpdateServiceItemRequest,
  UpdateServiceRequest,
  UpsertDiagnosticRequest,
} from './types';

export async function listServiceOrders(filter: ServiceFilter = {}): Promise<ApiEnvelope<ServiceOrderListItem[]>> {
  const params = new URLSearchParams();
  if (filter.q) params.set('q', filter.q);
  if (filter.status) params.set('status', filter.status);
  if (filter.technician_id) params.set('technician_id', filter.technician_id);
  if (filter.outlet_id) params.set('outlet_id', filter.outlet_id);
  if (filter.from) params.set('from', filter.from);
  if (filter.to) params.set('to', filter.to);
  if (filter.customer_id) params.set('customer_id', filter.customer_id);
  if (filter.page) params.set('page', String(filter.page));
  if (filter.per_page) params.set('per_page', String(filter.per_page));

  const query = params.toString();
  return api.getEnvelope<ServiceOrderListItem[]>(`/service-orders${query ? `?${query}` : ''}`);
}

export async function getServiceOrder(id: string): Promise<ServiceOrderDetail> {
  return api.get<ServiceOrderDetail>(`/service-orders/${id}`);
}

export async function createServiceOrder(payload: CreateServiceRequest): Promise<{ id: string; ticket_no: string; tracking_url: string }> {
  return api.post<{ id: string; ticket_no: string; tracking_url: string }>('/service-orders', payload);
}

export async function updateServiceOrder(id: string, payload: UpdateServiceRequest): Promise<ServiceOrderDetail> {
  return api.patch<ServiceOrderDetail>(`/service-orders/${id}`, payload);
}

export async function getServiceTimeline(id: string): Promise<StatusHistoryEntry[]> {
  return api.get<StatusHistoryEntry[]>(`/service-orders/${id}/timeline`);
}

export async function setCredential(
  orderId: string,
  payload: { type: string; value: string; reason?: string },
): Promise<{ message: string }> {
  return api.put<{ message: string }>(`/service-orders/${orderId}/credential`, payload);
}

export async function deleteCredential(orderId: string): Promise<{ message: string }> {
  return api.delete<{ message: string }>(`/service-orders/${orderId}/credential`);
}

export async function revealCredential(orderId: string, reason: string): Promise<{ type: string; value: string }> {
  return api.post<{ type: string; value: string }>(`/service-orders/${orderId}/credential/reveal`, { reason });
}

export async function uploadServicePhoto(
  orderId: string,
  file: File,
  stage = 'intake',
  isPublic = false,
): Promise<ServicePhoto> {
  const formData = new FormData();
  formData.append('file', file);
  formData.append('stage', stage);
  formData.append('is_public', isPublic ? 'true' : 'false');

  return apiRequest<ServicePhoto>(`/service-orders/${orderId}/photos`, {
    method: 'POST',
    body: formData,
  });
}

export async function deleteServicePhoto(orderId: string, photoId: string): Promise<{ message: string }> {
  return api.delete<{ message: string }>(`/service-orders/${orderId}/photos/${photoId}`);
}

export async function getTechnicians(outletId?: string): Promise<TechnicianOption[]> {
  const query = outletId ? `?outlet_id=${outletId}` : '';
  return api.get<TechnicianOption[]>(`/technicians${query}`);
}

export async function changeServiceStatus(
  orderId: string,
  payload: ChangeStatusRequest,
): Promise<ServiceOrderDetail> {
  return api.patch<ServiceOrderDetail>(`/service-orders/${orderId}/status`, payload);
}

export async function assignTechnician(
  orderId: string,
  payload: AssignTechnicianRequest,
): Promise<ServiceOrderDetail> {
  return api.post<ServiceOrderDetail>(`/service-orders/${orderId}/assign`, payload);
}

export async function getDiagnostic(orderId: string): Promise<Diagnostic | null> {
  return api.get<Diagnostic | null>(`/service-orders/${orderId}/diagnostic`);
}

export async function upsertDiagnostic(
  orderId: string,
  payload: UpsertDiagnosticRequest,
): Promise<Diagnostic> {
  return api.put<Diagnostic>(`/service-orders/${orderId}/diagnostic`, payload);
}

export async function listServiceItems(orderId: string): Promise<ServiceOrderItem[]> {
  return api.get<ServiceOrderItem[]>(`/service-orders/${orderId}/items`);
}

export async function addServiceItem(
  orderId: string,
  payload: AddServiceItemRequest,
): Promise<ServiceOrderItem> {
  return api.post<ServiceOrderItem>(`/service-orders/${orderId}/items`, payload);
}

export async function updateServiceItem(
  orderId: string,
  itemId: string,
  payload: UpdateServiceItemRequest,
): Promise<ServiceOrderItem> {
  return api.patch<ServiceOrderItem>(`/service-orders/${orderId}/items/${itemId}`, payload);
}

export async function deleteServiceItem(orderId: string, itemId: string): Promise<{ message: string }> {
  return api.delete<{ message: string }>(`/service-orders/${orderId}/items/${itemId}`);
}

export async function searchSpareparts(q: string, limit = 20): Promise<SparepartOption[]> {
  const params = new URLSearchParams();
  if (q.trim()) params.set('q', q.trim());
  params.set('limit', String(limit));
  return api.get<SparepartOption[]>(`/spareparts/search?${params.toString()}`);
}
