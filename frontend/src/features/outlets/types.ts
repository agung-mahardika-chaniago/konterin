export interface Outlet {
  id: string;
  code: string;
  name: string;
  address: string | null;
  phone: string | null;
  timezone: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateOutletRequest {
  code: string;
  name: string;
  address: string | null;
  phone: string | null;
  timezone: string;
  is_active: boolean;
}

export interface UpdateOutletRequest {
  code?: string;
  name?: string;
  address?: string | null;
  phone?: string | null;
  timezone?: string;
  is_active?: boolean;
}
