import type { AuthUser } from '@/types/auth';

export function hasPermission(user: AuthUser | null, permission: string): boolean {
  if (!user) return false;
  return user.permissions.includes(permission);
}

export function hasAnyPermission(user: AuthUser | null, permissions: string[]): boolean {
  if (!user) return false;
  return permissions.some((permission) => user.permissions.includes(permission));
}

/**
 * Where to send a user immediately after login or while restoring a session.
 */
export function primaryRouteFor(user: AuthUser): string {
  if (user.must_change_password) return '/change-password';
  if (user.role === 'super_admin') return '/admin';
  if (user.permissions.includes('dashboard.view')) return '/dashboard';
  return '/403';
}
