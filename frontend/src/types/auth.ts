export type Role = 'super_admin' | 'owner' | 'admin' | 'technician' | 'cashier';

export interface AuthOutlet {
  id: string;
  name: string;
  code: string;
}

export interface AuthUser {
  id: string;
  name: string;
  email: string;
  phone?: string | null;
  avatar_path?: string | null;
  role: Role;
  tenant_id: string | null;
  permissions: string[];
  outlets: AuthOutlet[];
  active_outlet_id: string | null;
  must_change_password: boolean;
  last_login_at?: string | null;
}

export interface AuthSession {
  access_token: string;
  token_type: string;
  expires_in: number;
  user: AuthUser;
}

export interface ProfileResponse {
  user: AuthUser;
}
