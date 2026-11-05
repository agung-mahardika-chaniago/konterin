import { Navigate, Route, Routes } from 'react-router-dom';

import { AppShell } from '@/components/layout/app-shell';
import { ChangePasswordPage } from '@/features/auth/change-password-page';
import { LoginPage } from '@/features/auth/login-page';
import { PermissionGuard } from '@/features/auth/permission-guard';
import { RequireAuth } from '@/features/auth/require-auth';
import { CustomerDetailPage } from '@/features/customers/pages/customer-detail-page';
import { CustomerListPage } from '@/features/customers/pages/customer-list-page';
import { DashboardPage } from '@/features/dashboard/dashboard-page';
import { ServiceCreatePage } from '@/features/services/pages/service-create-page';
import { ServiceDetailPage } from '@/features/services/pages/service-detail-page';
import { ServiceListPage } from '@/features/services/pages/service-list-page';
import { ForbiddenPage } from '@/pages/forbidden-page';
import { NotFoundPage } from '@/pages/not-found-page';
import { PlatformPage } from '@/pages/platform-page';
import { primaryRouteFor } from '@/lib/permissions';
import { useAuthStore } from '@/stores/auth-store';

function HomeRoute() {
  const user = useAuthStore((state) => state.user);
  if (!user) return null;
  return <Navigate to={primaryRouteFor(user)} replace />;
}

export function AppRouter() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route
        path="/change-password"
        element={
          <RequireAuth>
            <ChangePasswordPage />
          </RequireAuth>
        }
      />
      <Route
        path="/"
        element={
          <RequireAuth>
            <HomeRoute />
          </RequireAuth>
        }
      />
      <Route
        path="/dashboard"
        element={
          <RequireAuth>
            <PermissionGuard permission="dashboard.view">
              <AppShell>
                <DashboardPage />
              </AppShell>
            </PermissionGuard>
          </RequireAuth>
        }
      />
      <Route
        path="/services"
        element={
          <RequireAuth>
            <PermissionGuard permission="service.view">
              <AppShell>
                <ServiceListPage />
              </AppShell>
            </PermissionGuard>
          </RequireAuth>
        }
      />
      <Route
        path="/services/new"
        element={
          <RequireAuth>
            <PermissionGuard permission="service.create">
              <AppShell>
                <ServiceCreatePage />
              </AppShell>
            </PermissionGuard>
          </RequireAuth>
        }
      />
      <Route
        path="/services/:id"
        element={
          <RequireAuth>
            <PermissionGuard permission="service.view">
              <AppShell>
                <ServiceDetailPage />
              </AppShell>
            </PermissionGuard>
          </RequireAuth>
        }
      />
      <Route
        path="/customers"
        element={
          <RequireAuth>
            <PermissionGuard permission="customer.view">
              <AppShell>
                <CustomerListPage />
              </AppShell>
            </PermissionGuard>
          </RequireAuth>
        }
      />
      <Route
        path="/customers/:id"
        element={
          <RequireAuth>
            <PermissionGuard permission="customer.view">
              <AppShell>
                <CustomerDetailPage />
              </AppShell>
            </PermissionGuard>
          </RequireAuth>
        }
      />
      <Route
        path="/admin"
        element={
          <RequireAuth>
            <PermissionGuard permission="platform.tenant.view">
              <PlatformPage />
            </PermissionGuard>
          </RequireAuth>
        }
      />
      <Route path="/403" element={<ForbiddenPage />} />
      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  );
}
