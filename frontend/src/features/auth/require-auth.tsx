import type { ReactNode } from 'react';
import { Navigate, useLocation } from 'react-router-dom';

import { FullPageLoader } from '@/components/ui/spinner';
import { useAuthStore } from '@/stores/auth-store';

export function RequireAuth({ children }: { children: ReactNode }) {
  const status = useAuthStore((state) => state.status);
  const user = useAuthStore((state) => state.user);
  const location = useLocation();

  if (status === 'loading') {
    return <FullPageLoader label="Memeriksa sesi…" />;
  }
  if (status === 'unauthenticated' || !user) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />;
  }
  if (user.must_change_password && location.pathname !== '/change-password') {
    return <Navigate to="/change-password" replace />;
  }
  return <>{children}</>;
}
