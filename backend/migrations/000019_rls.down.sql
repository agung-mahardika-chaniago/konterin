-- Remove tenant isolation policies. The fixphone_app role is cluster-wide and
-- may be shared by other databases, so it is deliberately NOT dropped here;
-- only its privileges on this database are withdrawn.

ALTER TABLE notification_reads DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON notification_reads;

ALTER TABLE user_outlets DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON user_outlets;

ALTER TABLE refresh_tokens DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON refresh_tokens;

ALTER TABLE tenants DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON tenants;

ALTER TABLE whatsapp_messages DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON whatsapp_messages;

ALTER TABLE warranty_claims DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON warranty_claims;

ALTER TABLE warranties DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON warranties;

ALTER TABLE users DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON users;

ALTER TABLE suppliers DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON suppliers;

ALTER TABLE spareparts DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON spareparts;

ALTER TABLE sparepart_categories DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON sparepart_categories;

ALTER TABLE service_status_history DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON service_status_history;

ALTER TABLE service_photos DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON service_photos;

ALTER TABLE service_orders DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON service_orders;

ALTER TABLE service_order_items DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON service_order_items;

ALTER TABLE service_diagnostics DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON service_diagnostics;

ALTER TABLE payments DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON payments;

ALTER TABLE outlets DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON outlets;

ALTER TABLE notifications DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON notifications;

ALTER TABLE invoices DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON invoices;

ALTER TABLE invoice_items DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON invoice_items;

ALTER TABLE inventory_movements DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON inventory_movements;

ALTER TABLE expenses DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON expenses;

ALTER TABLE expense_categories DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON expense_categories;

ALTER TABLE document_sequences DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON document_sequences;

ALTER TABLE devices DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON devices;

ALTER TABLE customers DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON customers;

ALTER TABLE business_settings DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON business_settings;

ALTER TABLE backups DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON backups;

ALTER TABLE audit_logs DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON audit_logs;

ALTER DEFAULT PRIVILEGES IN SCHEMA public
    REVOKE SELECT, INSERT, UPDATE, DELETE ON TABLES FROM fixphone_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    REVOKE USAGE, SELECT ON SEQUENCES FROM fixphone_app;

REVOKE ALL ON ALL TABLES IN SCHEMA public FROM fixphone_app;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA public FROM fixphone_app;
REVOKE USAGE ON SCHEMA public FROM fixphone_app;
