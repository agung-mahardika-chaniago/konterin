package integration_test

import (
	"testing"

	"fixphone/backend/internal/database"
	"fixphone/backend/tests/testutil"

	"gorm.io/gorm"
)

func TestMigrationUpDownUp(t *testing.T) {
	db := freshDB(t)

	t.Run("schema shape", func(t *testing.T) {
		var tableCount int
		if err := db.Raw(`SELECT count(*) FROM information_schema.tables
			WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
			  AND table_name <> 'schema_migrations'`).Scan(&tableCount).Error; err != nil {
			t.Fatalf("count tables: %v", err)
		}
		if tableCount != 32 {
			t.Fatalf("table count = %d, want 32", tableCount)
		}

		var triggerCount int
		if err := db.Raw(`SELECT count(*) FROM information_schema.triggers
			WHERE trigger_schema = 'public' AND trigger_name LIKE 'trg_%_updated_at'`).Scan(&triggerCount).Error; err != nil {
			t.Fatalf("count triggers: %v", err)
		}
		if triggerCount != 17 {
			t.Fatalf("updated_at trigger count = %d, want 17", triggerCount)
		}

		var indexCount int
		if err := db.Raw(`SELECT count(*) FROM pg_indexes
			WHERE schemaname = 'public' AND indexname IN (
				'uq_customers_tenant_phone_active',
				'idx_customers_name_trgm',
				'idx_spareparts_low_stock'
			)`).Scan(&indexCount).Error; err != nil {
			t.Fatalf("count indexes: %v", err)
		}
		if indexCount != 3 {
			t.Fatalf("critical index count = %d, want 3", indexCount)
		}

		var settingsColumnExists bool
		if err := db.Raw(`SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = 'business_settings'
			  AND column_name = 'technician_sees_only_assigned'
		)`).Scan(&settingsColumnExists).Error; err != nil {
			t.Fatalf("check settings column: %v", err)
		}
		if !settingsColumnExists {
			t.Fatal("business_settings.technician_sees_only_assigned is missing")
		}
	})

	t.Run("down and up again", func(t *testing.T) {
		migrator, err := database.NewMigrator(testutil.DSN())
		if err != nil {
			t.Fatalf("new migrator: %v", err)
		}
		defer migrator.Close()

		if err := migrator.Down(0); err != nil {
			t.Fatalf("migrate down: %v", err)
		}
		version, dirty, err := migrator.Status()
		if err != nil {
			t.Fatalf("status after down: %v", err)
		}
		if version != 0 || dirty {
			t.Fatalf("after down version=%d dirty=%t, want 0/false", version, dirty)
		}

		var tenantsExists bool
		if err := db.Raw(`SELECT to_regclass('public.tenants') IS NOT NULL`).Scan(&tenantsExists).Error; err != nil {
			t.Fatalf("check tenants table: %v", err)
		}
		if tenantsExists {
			t.Fatal("tenants table still exists after full down")
		}

		if err := migrator.Up(); err != nil {
			t.Fatalf("migrate up again: %v", err)
		}
		version, dirty, err = migrator.Status()
		if err != nil {
			t.Fatalf("status after second up: %v", err)
		}
		if version != database.ExpectedVersion || dirty {
			t.Fatalf("after second up version=%d dirty=%t, want %d/false", version, dirty, database.ExpectedVersion)
		}
	})
}

func TestCriticalConstraints(t *testing.T) {
	db := freshDB(t)
	seedSystem(t, db)

	tenantA := createTenant(t, db, "Toko Alpha", "toko-alpha", "owner.alpha@example.com")
	tenantB := createTenant(t, db, "Toko Beta", "toko-beta", "owner.beta@example.com")

	customerA := insertCustomer(t, db, tenantA.TenantID, "Budi", "081234567890")
	customerB := insertCustomer(t, db, tenantB.TenantID, "Siti", "081234567890")

	t.Run("negative stock is rejected", func(t *testing.T) {
		err := db.Exec(`INSERT INTO spareparts (tenant_id, sku, name, stock, min_stock, cost_price, sell_price)
			VALUES (?, 'SKU-NEG', 'LCD Retak', -1, 0, 0, 0)`, tenantA.TenantID).Error
		assertConstraintViolation(t, err, "chk_spareparts_stock")
	})

	t.Run("illegal service status is rejected", func(t *testing.T) {
		err := db.Exec(`INSERT INTO service_orders
			(tenant_id, outlet_id, ticket_no, customer_id, status, device_brand, device_model, complaint)
			VALUES (?, ?, 'FP-BAD-001', ?, 'bogus', 'Samsung', 'A52', 'Mati total')`,
			tenantA.TenantID, tenantA.OutletID, customerA).Error
		assertConstraintViolation(t, err, "chk_service_orders_status")
	})

	t.Run("credential pair is enforced", func(t *testing.T) {
		err := db.Exec(`INSERT INTO service_orders
			(tenant_id, outlet_id, ticket_no, customer_id, status, device_brand, device_model, complaint, credential_type)
			VALUES (?, ?, 'FP-BAD-002', ?, 'received', 'Samsung', 'A52', 'Mati total', 'pin')`,
			tenantA.TenantID, tenantA.OutletID, customerA).Error
		assertConstraintViolation(t, err, "chk_service_orders_credential_pair")
	})

	t.Run("duplicate ticket is rejected per tenant", func(t *testing.T) {
		ticket := "FP-20261103-0001"
		if err := db.Exec(`INSERT INTO service_orders
			(tenant_id, outlet_id, ticket_no, customer_id, status, device_brand, device_model, complaint)
			VALUES (?, ?, ?, ?, 'received', 'Xiaomi', 'Redmi Note 12', 'Layar pecah')`,
			tenantA.TenantID, tenantA.OutletID, ticket, customerA).Error; err != nil {
			t.Fatalf("insert first ticket: %v", err)
		}
		err := db.Exec(`INSERT INTO service_orders
			(tenant_id, outlet_id, ticket_no, customer_id, status, device_brand, device_model, complaint)
			VALUES (?, ?, ?, ?, 'received', 'Xiaomi', 'Redmi Note 12', 'Layar pecah')`,
			tenantA.TenantID, tenantA.OutletID, ticket, customerA).Error
		assertConstraintViolation(t, err, "uq_service_orders_tenant_ticket")

		if err := db.Exec(`INSERT INTO service_orders
			(tenant_id, outlet_id, ticket_no, customer_id, status, device_brand, device_model, complaint)
			VALUES (?, ?, ?, ?, 'received', 'Oppo', 'A78', 'Baterai drop')`,
			tenantB.TenantID, tenantB.OutletID, ticket, customerB).Error; err != nil {
			t.Fatalf("same ticket in another tenant should be allowed: %v", err)
		}
	})

	t.Run("duplicate customer phone is scoped per tenant", func(t *testing.T) {
		err := db.Exec(`INSERT INTO customers (tenant_id, name, phone, phone_normalized)
			VALUES (?, 'Budi Kembar', '081234567890', '081234567890')`, tenantA.TenantID).Error
		assertConstraintViolation(t, err, "uq_customers_tenant_phone_active")
	})

	t.Run("overpaid invoice is rejected", func(t *testing.T) {
		var orderID string
		if err := db.Raw(`INSERT INTO service_orders
			(tenant_id, outlet_id, ticket_no, customer_id, status, device_brand, device_model, complaint)
			VALUES (?, ?, 'FP-20261103-0002', ?, 'completed', 'Asus', 'Zenfone', 'Tidak menyala')
			RETURNING id`, tenantA.TenantID, tenantA.OutletID, customerA).Scan(&orderID).Error; err != nil {
			t.Fatalf("insert order for invoice: %v", err)
		}
		err := db.Exec(`INSERT INTO invoices
			(tenant_id, outlet_id, service_order_id, invoice_no, customer_id, subtotal_service, total_amount, paid_amount)
			VALUES (?, ?, ?, 'INV-BAD-001', ?, 100000, 100000, 150000)`,
			tenantA.TenantID, tenantA.OutletID, orderID, customerA).Error
		assertConstraintViolation(t, err, "chk_invoices_paid_not_over_total")
	})

	t.Run("warranty end date cannot precede start", func(t *testing.T) {
		var orderID string
		if err := db.Raw(`INSERT INTO service_orders
			(tenant_id, outlet_id, ticket_no, customer_id, status, device_brand, device_model, complaint)
			VALUES (?, ?, 'FP-20261103-0003', ?, 'completed', 'Vivo', 'Y36', 'Speaker rusak')
			RETURNING id`, tenantA.TenantID, tenantA.OutletID, customerA).Scan(&orderID).Error; err != nil {
			t.Fatalf("insert order for warranty: %v", err)
		}
		var invoiceID string
		if err := db.Raw(`INSERT INTO invoices
			(tenant_id, outlet_id, service_order_id, invoice_no, customer_id, subtotal_service, total_amount)
			VALUES (?, ?, ?, 'INV-OK-001', ?, 50000, 50000)
			RETURNING id`, tenantA.TenantID, tenantA.OutletID, orderID, customerA).Scan(&invoiceID).Error; err != nil {
			t.Fatalf("insert invoice for warranty: %v", err)
		}
		err := db.Exec(`INSERT INTO warranties
			(tenant_id, outlet_id, service_order_id, invoice_id, customer_id, warranty_no,
			 type, start_date, end_date, days)
			VALUES (?, ?, ?, ?, ?, 'WR-BAD-001', 'service', '2026-11-10', '2026-11-01', 30)`,
			tenantA.TenantID, tenantA.OutletID, orderID, invoiceID, customerA).Error
		assertConstraintViolation(t, err, "chk_warranties_dates")
	})

	t.Run("audit log is append only", func(t *testing.T) {
		if err := db.Exec(`INSERT INTO audit_logs (tenant_id, action, module, entity_type)
			VALUES (?, 'test.create', 'test', 'test')`, tenantA.TenantID).Error; err != nil {
			t.Fatalf("insert audit log: %v", err)
		}
		var exists bool
		if err := db.Raw(`SELECT EXISTS (SELECT 1 FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = 'audit_logs' AND column_name = 'updated_at')`).Scan(&exists).Error; err != nil {
			t.Fatalf("inspect audit_logs: %v", err)
		}
		if exists {
			t.Fatal("audit_logs must not have an updated_at column")
		}
	})
}

func insertCustomer(t *testing.T, db *gorm.DB, tenantID, name, phone string) string {
	t.Helper()
	var id string
	if err := db.Raw(`INSERT INTO customers (tenant_id, name, phone, phone_normalized)
		VALUES (?, ?, ?, ?) RETURNING id`, tenantID, name, phone, phone).Scan(&id).Error; err != nil {
		t.Fatalf("insert customer: %v", err)
	}
	return id
}
