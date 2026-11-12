export interface UserOutlet {
  id: string;
  code: string;
  name: string;
}

export interface User {
  id: string;
  name: string;
  email: string;
  phone: string | null;
  role: string;
  role_name: string;
  is_active: boolean;
  must_change_password: boolean;
  last_login_at: string | null;
  outlets: UserOutlet[];
  created_at: string;
  updated_at: string;
}

export interface RoleInfo {
  id: string;
  slug: string;
  name: string;
  permissions: string[];
}

export interface UserFilter {
  page?: number;
  per_page?: number;
  q?: string;
  role?: string;
  status?: string;
}

export interface UserListData {
  items: User[];
}

export interface CreateUserRequest {
  name: string;
  email: string;
  phone: string | null;
  role: string;
  password: string;
  outlet_ids: string[];
  is_active: boolean;
}

export interface UpdateUserRequest {
  name?: string;
  phone?: string | null;
  role?: string;
  is_active?: boolean;
}

export interface ResetPasswordResult {
  temporary_password: string;
  must_change_password: boolean;
}
