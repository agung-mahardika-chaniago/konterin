import { api, apiRequest } from '@/lib/api-client';
import type { ApiEnvelope } from '@/types/api';
import type {
  CreateServiceRequest,
  ServiceFilter,
  ServiceOrderDetail,
  ServiceOrderListItem,
  ServicePhoto,
  StatusHistoryEntry,
  TechnicianOption,
  UpdateServiceRequest,
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
