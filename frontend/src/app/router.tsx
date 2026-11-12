import { Navigate, Route, Routes } from 'react-router-dom';

import { AppShell } from '@/components/layout/app-shell';
import { AuditLogPage } from '@/features/audit/pages/audit-log-page';
import { ChangePasswordPage } from '@/features/auth/change-password-page';
import { LoginPage } from '@/features/auth/login-page';
import { PermissionGuard } from '@/features/auth/permission-guard';
import { RequireAuth } from '@/features/auth/require-auth';
import { CustomerDetailPage } from '@/features/customers/pages/customer-detail-page';
import { CustomerListPage } from '@/features/customers/pages/customer-list-page';
import { DashboardPage } from '@/features/dashboard/dashboard-page';
import { ExpenseListPage } from '@/features/expenses/pages/expense-list-page';
import { MovementListPage } from '@/features/inventory/pages/movement-list-page';
import { PurchasePage } from '@/features/inventory/pages/purchase-page';
import { SparepartDetailPage } from '@/features/inventory/pages/sparepart-detail-page';
import { SparepartListPage } from '@/features/inventory/pages/sparepart-list-page';
import { CashierPage } from '@/features/invoices/pages/cashier-page';
import { InvoiceDetailPage } from '@/features/invoices/pages/invoice-detail-page';
import { InvoiceListPage } from '@/features/invoices/pages/invoice-list-page';
import { NotificationListPage } from '@/features/notifications/pages/notification-list-page';
import { WhatsAppOutboxPage } from '@/features/notifications/pages/whatsapp-outbox-page';
import { OutletListPage } from '@/features/outlets/pages/outlet-list-page';
import { ReportsPage } from '@/features/reports/pages/reports-page';
import { ServiceCreatePage } from '@/features/services/pages/service-create-page';
import { ServiceDetailPage } from '@/features/services/pages/service-detail-page';
import { ServiceListPage } from '@/features/services/pages/service-list-page';
import { SettingsPage } from '@/features/settings/pages/settings-page';
import { TrackingDetailPage } from '@/features/tracking/pages/tracking-detail-page';
import { TrackingLookupPage } from '@/features/tracking/pages/tracking-lookup-page';
import { UserListPage } from '@/features/users/pages/user-list-page';
import { ClaimDetailPage } from '@/features/warranties/pages/claim-detail-page';
import { WarrantyDetailPage } from '@/features/warranties/pages/warranty-detail-page';
import { WarrantyListPage } from '@/features/warranties/pages/warranty-list-page';
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
      <Route path="/track" element={<TrackingLookupPage />} />
      <Route path="/track/:token" element={<TrackingDetailPage />} />
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
        path="/spareparts"
        element={
          <RequireAuth>
            <PermissionGuard permission="sparepart.view">
              <AppShell>
                <SparepartListPage />
              </AppShell>
            </PermissionGuard>
          </RequireAuth>
        }
      />
      <Route
        path="/spareparts/:id"
        element={
          <RequireAuth>
            <PermissionGuard permission="sparepart.view">
              <AppShell>
                <SparepartDetailPage />
              </AppShell>
            </PermissionGuard>
          </RequireAuth>
        }
      />
      <Route
        path="/inventory/movements"
        element={
          <RequireAuth>
            <PermissionGuard permission="inventory.view">
              <AppShell>
                <MovementListPage />
              </AppShell>
            </PermissionGuard>
          </RequireAuth>
        }
      />
      <Route
        path="/inventory/purchases/new"
        element={
          <RequireAuth>
            <PermissionGuard permission="inventory.purchase">
              <AppShell>
                <PurchasePage />
              </AppShell>
            </PermissionGuard>
          </RequireAuth>
        }
      />
      <Route
        path="/cashier"
        element={
          <RequireAuth>
            <PermissionGuard permission="payment.view">
              <AppShell>
                <CashierPage />
              </AppShell>
            </PermissionGuard>
          </RequireAuth>
        }
      />
      <Route
        path="/invoices"
        element={
          <RequireAuth>
            <PermissionGuard permission="invoice.view">
              <AppShell>
                <InvoiceListPage />
              </AppShell>
            </PermissionGuard>
          </RequireAuth>
        }
      />
      <Route
        path="/invoices/:id"
        element={
          <RequireAuth>
            <PermissionGuard permission="invoice.view">
              <AppShell>
                <InvoiceDetailPage />
              </AppShell>
            </PermissionGuard>
          </RequireAuth>
        }
      />
      <Route
        path="/expenses"
        element={
          <RequireAuth>
            <PermissionGuard permission="expense.view">
              <AppShell>
                <ExpenseListPage />
              </AppShell>
            </PermissionGuard>
          </RequireAuth>
        }
      />
      <Route
        path="/reports"
        element={
          <RequireAuth>
            <PermissionGuard permission="report.view">
              <AppShell>
                <ReportsPage />
              </AppShell>
            </PermissionGuard>
          </RequireAuth>
        }
      />
      <Route
        path="/users"
        element={
          <RequireAuth>
            <PermissionGuard permission="user.view">
              <AppShell>
                <UserListPage />
              </AppShell>
            </PermissionGuard>
          </RequireAuth>
        }
      />
      <Route
        path="/outlets"
        element={
          <RequireAuth>
            <PermissionGuard permission="outlet.view">
              <AppShell>
                <OutletListPage />
              </AppShell>
            </PermissionGuard>
          </RequireAuth>
        }
      />
      <Route
        path="/settings"
        element={
          <RequireAuth>
            <PermissionGuard permission="settings.view">
              <AppShell>
                <SettingsPage />
              </AppShell>
            </PermissionGuard>
          </RequireAuth>
        }
      />
      <Route
        path="/audit-logs"
        element={
          <RequireAuth>
            <PermissionGuard permission="audit.view">
              <AppShell>
                <AuditLogPage />
              </AppShell>
            </PermissionGuard>
          </RequireAuth>
        }
      />
      <Route
        path="/warranties"
        element={
          <RequireAuth>
            <PermissionGuard permission="warranty.view">
              <AppShell>
                <WarrantyListPage />
              </AppShell>
            </PermissionGuard>
          </RequireAuth>
        }
      />
      <Route
        path="/warranties/:id"
        element={
          <RequireAuth>
            <PermissionGuard permission="warranty.view">
              <AppShell>
                <WarrantyDetailPage />
              </AppShell>
            </PermissionGuard>
          </RequireAuth>
        }
      />
      <Route
        path="/claims/:id"
        element={
          <RequireAuth>
            <PermissionGuard permission="warranty.view">
              <AppShell>
                <ClaimDetailPage />
              </AppShell>
            </PermissionGuard>
          </RequireAuth>
        }
      />
      <Route
        path="/notifications"
        element={
          <RequireAuth>
            <AppShell>
              <NotificationListPage />
            </AppShell>
          </RequireAuth>
        }
      />
      <Route
        path="/whatsapp-outbox"
        element={
          <RequireAuth>
            <PermissionGuard permission="settings.manage">
              <AppShell>
                <WhatsAppOutboxPage />
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
