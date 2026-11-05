import type { ReactNode } from 'react';
import { Navigate } from 'react-router-dom';

import { hasPermission } from '@/lib/permissions';
import { useAuthStore } from '@/stores/auth-store';

interface PermissionGuardProps {
  permission: string;
  children: ReactNode;
}

/**
 * Default-deny UI guard. The server still enforces every permission; this only
 * prevents users from landing on screens they cannot use.
 */
export function PermissionGuard({ permission, children }: PermissionGuardProps) {
  const user = useAuthStore((state) => state.user);
  if (!hasPermission(user, permission)) {
    return <Navigate to="/403" replace />;
  }
  return <>{children}</>;
}
