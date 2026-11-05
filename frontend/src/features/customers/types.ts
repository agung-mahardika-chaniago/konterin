export interface Device {
  id: string;
  brand: string;
  model: string;
  imei?: string | null;
  color?: string | null;
  notes?: string | null;
  created_at: string;
  updated_at: string;
}

export interface CustomerStats {
  total_services: number;
  total_paid: number;
  last_service_at: string | null;
}

export interface Customer {
  id: string;
  name: string;
  phone: string;
  address?: string | null;
  notes?: string | null;
  last_service_at?: string | null;
  created_at: string;
  updated_at: string;
  devices?: Device[];
  stats?: CustomerStats;
}

export interface CustomerFilter {
  q?: string;
  page?: number;
  per_page?: number;
}

export interface CreateCustomerPayload {
  name: string;
  phone: string;
  address?: string;
  notes?: string;
}

export interface UpdateCustomerPayload {
  name?: string;
  phone?: string;
  address?: string;
  notes?: string;
}

export interface CreateDevicePayload {
  brand: string;
  model: string;
  imei?: string;
  color?: string;
  notes?: string;
}
