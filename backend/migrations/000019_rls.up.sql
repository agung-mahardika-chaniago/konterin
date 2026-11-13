-- Row Level Security as the last line of defense for tenant isolation.
--
-- Runtime wiring (SET LOCAL app.tenant_id per request inside a transaction) is
-- intentionally NOT part of this migration: the application still connects as
-- the database owner until Tahap 12. This migration prepares the database so a
-- non-superuser application role can enforce the policies once the wiring is
-- completed. See docs/07-keamanan.md §5 and docs/18-implementasi-tahap-11.md.
--
-- Every policy fails closed: when app.tenant_id is unset or empty the
-- comparison against NULL is never true, so no rows are visible and INSERTs
-- are rejected.

-- The application role never owns tables and can neither bypass nor create
-- database objects. LOGIN/PASSWORD are managed by the DBA (production) or by
-- the RLS isolation test (disposable test database). The role is cluster-wide,
-- so this migration must never DROP it.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'fixphone_app') THEN
        CREATE ROLE fixphone_app NOLOGIN NOSUPERUSER NOBYPASSRLS NOCREATEDB NOCREATEROLE;
    END IF;
END
$$;

ALTER ROLE fixphone_app NOSUPERUSER NOBYPASSRLS NOCREATEDB NOCREATEROLE;

GRANT USAGE ON SCHEMA public TO fixphone_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO fixphone_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO fixphone_app;

-- Keep future tables reachable for the application role. The migration runner
-- creates the objects, so default privileges are registered for it.
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO fixphone_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT USAGE, SELECT ON SEQUENCES TO fixphone_app;

-- Audit logs are append-only (docs/07 §10): the application role must not be
-- able to rewrite history.
REVOKE UPDATE, DELETE ON audit_logs FROM fixphone_app;

-- Tables with a direct tenant_id column.
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON audit_logs
    USING (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE backups ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON backups
    USING (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE business_settings ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON business_settings
    USING (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE customers ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON customers
    USING (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE devices ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON devices
    USING (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE document_sequences ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON document_sequences
    USING (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE expense_categories ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON expense_categories
    USING (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE expenses ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON expenses
    USING (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE inventory_movements ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON inventory_movements
    USING (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE invoice_items ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON invoice_items
    USING (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE invoices ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON invoices
    USING (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE notifications ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON notifications
    USING (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE outlets ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON outlets
    USING (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE payments ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON payments
    USING (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE service_diagnostics ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON service_diagnostics
    USING (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE service_order_items ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON service_order_items
    USING (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE service_orders ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON service_orders
    USING (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE service_photos ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON service_photos
    USING (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE service_status_history ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON service_status_history
    USING (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE sparepart_categories ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON sparepart_categories
    USING (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE spareparts ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON spareparts
    USING (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE suppliers ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON suppliers
    USING (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE users ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON users
    USING (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE warranties ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON warranties
    USING (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE warranty_claims ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON warranty_claims
    USING (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE whatsapp_messages ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON whatsapp_messages
    USING (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid);

-- tenants: a tenant row is visible only when it is the active tenant.
ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON tenants
    USING (id = nullif(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (id = nullif(current_setting('app.tenant_id', true), '')::uuid);

-- Tables without tenant_id inherit scope through their parent row. Each policy
-- subquery is itself filtered by the parent table policy above.
ALTER TABLE refresh_tokens ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON refresh_tokens
    USING (user_id IN (
        SELECT id FROM users
        WHERE tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid
    ))
    WITH CHECK (user_id IN (
        SELECT id FROM users
        WHERE tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid
    ));

ALTER TABLE user_outlets ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON user_outlets
    USING (user_id IN (
        SELECT id FROM users
        WHERE tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid
    ))
    WITH CHECK (user_id IN (
        SELECT id FROM users
        WHERE tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid
    ));

ALTER TABLE notification_reads ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON notification_reads
    USING (notification_id IN (
        SELECT id FROM notifications
        WHERE tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid
    ))
    WITH CHECK (notification_id IN (
        SELECT id FROM notifications
        WHERE tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid
    ));

-- roles, permissions and role_permissions hold global reference data and are
-- intentionally left without a tenant policy.
