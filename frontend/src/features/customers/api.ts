import { api } from '@/lib/api-client';
import type { ApiEnvelope } from '@/types/api';
import type {
  CreateCustomerPayload,
  CreateDevicePayload,
  Customer,
  CustomerFilter,
  CustomerStats,
  Device,
  UpdateCustomerPayload,
} from './types';
import type { ServiceOrderListItem } from '../services/types';

export async function listCustomers(filter: CustomerFilter = {}): Promise<ApiEnvelope<Customer[]>> {
  const params = new URLSearchParams();
  if (filter.q) params.set('q', filter.q);
  if (filter.page) params.set('page', String(filter.page));
  if (filter.per_page) params.set('per_page', String(filter.per_page));

  const query = params.toString();
  return api.getEnvelope<Customer[]>(`/customers${query ? `?${query}` : ''}`);
}

export async function getCustomer(id: string): Promise<Customer> {
  return api.get<Customer>(`/customers/${id}`);
}

export async function createCustomer(payload: CreateCustomerPayload): Promise<Customer> {
  return api.post<Customer>('/customers', payload);
}

export async function updateCustomer(id: string, payload: UpdateCustomerPayload): Promise<Customer> {
  return api.patch<Customer>(`/customers/${id}`, payload);
}

export async function deleteCustomer(id: string): Promise<{ message: string }> {
  return api.delete<{ message: string }>(`/customers/${id}`);
}

export async function listCustomerDevices(customerId: string): Promise<Device[]> {
  return api.get<Device[]>(`/customers/${customerId}/devices`);
}

export async function createCustomerDevice(customerId: string, payload: CreateDevicePayload): Promise<Device> {
  return api.post<Device>(`/customers/${customerId}/devices`, payload);
}

export async function listCustomerServices(customerId: string, page = 1, perPage = 20): Promise<ApiEnvelope<ServiceOrderListItem[]>> {
  return api.getEnvelope<ServiceOrderListItem[]>(`/customers/${customerId}/services?page=${page}&per_page=${perPage}`);
}

export async function getCustomerStats(customerId: string): Promise<CustomerStats> {
  return api.get<CustomerStats>(`/customers/${customerId}/stats`);
}
