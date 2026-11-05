-- Secondary indexes, partial uniques, and updated_at triggers.
-- Kept in a dedicated migration so query tuning has one review surface.

-- Identity / access
CREATE INDEX idx_outlets_tenant_active ON outlets (tenant_id, is_active);
CREATE INDEX idx_users_tenant_active ON users (tenant_id, is_active);
CREATE INDEX idx_users_tenant_role ON users (tenant_id, role_id);
CREATE UNIQUE INDEX uq_users_email_active ON users (email) WHERE deleted_at IS NULL;
CREATE INDEX idx_user_outlets_outlet ON user_outlets (outlet_id);
CREATE INDEX idx_refresh_tokens_user ON refresh_tokens (user_id);
CREATE INDEX idx_refresh_tokens_family ON refresh_tokens (family_id);
CREATE INDEX idx_refresh_tokens_active_expiry ON refresh_tokens (expires_at) WHERE revoked_at IS NULL;

-- Customers / devices
CREATE INDEX idx_customers_name_trgm ON customers USING gin (name gin_trgm_ops);
CREATE UNIQUE INDEX uq_customers_tenant_phone_active
    ON customers (tenant_id, phone_normalized) WHERE deleted_at IS NULL;
CREATE INDEX idx_customers_tenant_last_service ON customers (tenant_id, last_service_at DESC);
CREATE INDEX idx_devices_tenant_customer ON devices (tenant_id, customer_id);
CREATE UNIQUE INDEX uq_devices_tenant_imei_active
    ON devices (tenant_id, imei) WHERE imei IS NOT NULL AND deleted_at IS NULL;

-- Inventory
CREATE INDEX idx_suppliers_tenant_name ON suppliers (tenant_id, name);
CREATE UNIQUE INDEX uq_spareparts_tenant_sku_active
    ON spareparts (tenant_id, sku) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX uq_spareparts_tenant_barcode_active
    ON spareparts (tenant_id, barcode) WHERE barcode IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_spareparts_tenant_active_name ON spareparts (tenant_id, is_active, name);
CREATE INDEX idx_spareparts_tenant_category ON spareparts (tenant_id, category_id);
CREATE INDEX idx_spareparts_low_stock
    ON spareparts (tenant_id) WHERE stock <= min_stock AND is_active AND deleted_at IS NULL;
CREATE INDEX idx_inventory_movements_part_time
    ON inventory_movements (tenant_id, sparepart_id, created_at DESC);
CREATE INDEX idx_inventory_movements_reference ON inventory_movements (reference_type, reference_id);

-- Service
CREATE INDEX idx_service_orders_outlet_status
    ON service_orders (tenant_id, outlet_id, status, received_at DESC);
CREATE INDEX idx_service_orders_customer_time
    ON service_orders (tenant_id, customer_id, received_at DESC);
CREATE INDEX idx_service_orders_technician_status
    ON service_orders (tenant_id, assigned_technician_id, status);
CREATE INDEX idx_service_orders_estimated_done
    ON service_orders (tenant_id, estimated_done_at)
    WHERE status NOT IN ('picked_up', 'cancelled');
CREATE INDEX idx_service_orders_ready_pickup
    ON service_orders (tenant_id, picked_up_at) WHERE status = 'ready_for_pickup';
CREATE INDEX idx_service_order_items_order ON service_order_items (service_order_id);
CREATE INDEX idx_service_order_items_tenant_part ON service_order_items (tenant_id, sparepart_id);
CREATE INDEX idx_service_diagnostics_tenant_order ON service_diagnostics (tenant_id, service_order_id);
CREATE INDEX idx_service_status_history_order_time ON service_status_history (service_order_id, created_at);
CREATE INDEX idx_service_photos_order_stage ON service_photos (service_order_id, stage);

-- Finance
CREATE INDEX idx_invoices_outlet_status_issued
    ON invoices (tenant_id, outlet_id, status, issued_at DESC);
CREATE INDEX idx_invoices_tenant_customer ON invoices (tenant_id, customer_id);
CREATE INDEX idx_invoice_items_invoice ON invoice_items (invoice_id);
CREATE INDEX idx_payments_outlet_paid_posted
    ON payments (tenant_id, outlet_id, paid_at DESC) WHERE status = 'posted';
CREATE INDEX idx_payments_invoice_status ON payments (invoice_id, status);
CREATE UNIQUE INDEX uq_payments_reference_posted
    ON payments (invoice_id, reference_no) WHERE reference_no IS NOT NULL AND status = 'posted';
CREATE INDEX idx_expenses_outlet_date
    ON expenses (tenant_id, outlet_id, expense_date DESC) WHERE deleted_at IS NULL;

-- After-sales / notifications
CREATE INDEX idx_warranties_active_end ON warranties (tenant_id, end_date) WHERE status = 'active';
CREATE INDEX idx_warranties_customer ON warranties (tenant_id, customer_id);
CREATE INDEX idx_warranty_claims_tenant_status ON warranty_claims (tenant_id, status);
CREATE INDEX idx_warranty_claims_warranty ON warranty_claims (warranty_id);
CREATE INDEX idx_notifications_user_read_time ON notifications (user_id, is_read, created_at DESC);
CREATE INDEX idx_whatsapp_messages_queue ON whatsapp_messages (status, scheduled_at) WHERE status = 'queued';
CREATE INDEX idx_whatsapp_messages_tenant_created ON whatsapp_messages (tenant_id, created_at DESC);

-- Audit
CREATE INDEX idx_audit_logs_tenant_time ON audit_logs (tenant_id, created_at DESC);
CREATE INDEX idx_audit_logs_entity ON audit_logs (entity_type, entity_id);
CREATE INDEX idx_audit_logs_user_time ON audit_logs (user_id, created_at DESC);

-- updated_at triggers
CREATE TRIGGER trg_tenants_updated_at BEFORE UPDATE ON tenants
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_outlets_updated_at BEFORE UPDATE ON outlets
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_business_settings_updated_at BEFORE UPDATE ON business_settings
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_customers_updated_at BEFORE UPDATE ON customers
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_devices_updated_at BEFORE UPDATE ON devices
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_sparepart_categories_updated_at BEFORE UPDATE ON sparepart_categories
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_suppliers_updated_at BEFORE UPDATE ON suppliers
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_spareparts_updated_at BEFORE UPDATE ON spareparts
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_service_orders_updated_at BEFORE UPDATE ON service_orders
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_service_order_items_updated_at BEFORE UPDATE ON service_order_items
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_service_diagnostics_updated_at BEFORE UPDATE ON service_diagnostics
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_invoices_updated_at BEFORE UPDATE ON invoices
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_expense_categories_updated_at BEFORE UPDATE ON expense_categories
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_expenses_updated_at BEFORE UPDATE ON expenses
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_warranties_updated_at BEFORE UPDATE ON warranties
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_warranty_claims_updated_at BEFORE UPDATE ON warranty_claims
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
