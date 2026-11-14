package integration_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"fixphone/backend/internal/database"
	"fixphone/backend/tests/testutil"

	"gorm.io/gorm"
)

// rlsGormApp opens a GORM handle as the non-superuser application role and
// installs the production tenant binding callbacks. It exercises the same code
// path the API uses: shared repository handles whose statements are rewired to
// the request-scoped transaction carried by the context.
func rlsGormApp(t *testing.T, admin *sql.DB) *gorm.DB {
	t.Helper()

	// rlsOpenApp enables LOGIN for the role and proves the credentials work.
	rlsOpenApp(t, admin)

	u, err := url.Parse(testutil.DSN())
	if err != nil {
		t.Fatalf("parse test DSN: %v", err)
	}
	u.User = url.UserPassword(rlsAppRole, rlsAppPassword)

	app, err := database.Open(context.Background(), u.String(), database.DefaultOpenOptions())
	if err != nil {
		t.Fatalf("open GORM app role connection: %v", err)
	}
	t.Cleanup(func() { _ = database.Close(app) })

	if err := database.EnableTenantBinding(app); err != nil {
		t.Fatalf("enable tenant binding: %v", err)
	}
	return app
}

// rlsRequestTx mirrors middleware.Tenant: start a transaction on the
// application pool, set the tenant for RLS and attach the transaction to the
// request context.
func rlsRequestTx(t *testing.T, app *gorm.DB, tenantID string) (context.Context, *gorm.DB) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	tx := app.WithContext(ctx).Begin()
	if tx.Error != nil {
		t.Fatalf("begin request tx: %v", tx.Error)
	}
	if err := tx.Exec(`SELECT set_config('app.tenant_id', ?, true)`, tenantID).Error; err != nil {
		_ = tx.Rollback()
		t.Fatalf("set app.tenant_id: %v", err)
	}
	if err := tx.Exec(`SET LOCAL statement_timeout = '15s'`).Error; err != nil {
		_ = tx.Rollback()
		t.Fatalf("set statement_timeout: %v", err)
	}
	return database.ContextWithTx(ctx, tx), tx
}

func TestRLSGormBindingRequiresTenantContext(t *testing.T) {
	db := freshDB(t)
	seedSystem(t, db)
	alpha := createTenant(t, db, "Toko Alpha", "toko-alpha", "owner-alpha@fixphone.test")
	createTenant(t, db, "Toko Beta", "toko-beta", "owner-beta@fixphone.test")

	app := rlsGormApp(t, testutil.SQLDB(t, db))

	// A shared handle without a request transaction runs on a pooled
	// connection with no app.tenant_id, so every policy must fail closed.
	var count int64
	if err := app.Raw(`SELECT count(*) FROM tenants`).Scan(&count).Error; err != nil {
		t.Fatalf("count tenants without tenant context: %v", err)
	}
	if count != 0 {
		t.Fatalf("tenants visible without a tenant context = %d, want 0", count)
	}

	ctx, tx := rlsRequestTx(t, app, alpha.TenantID)
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback().Error
		}
	}()

	// The same shared handle, now carrying the request context, is rewired to
	// the request transaction and sees exactly one tenant.
	if err := app.WithContext(ctx).Raw(`SELECT count(*) FROM tenants`).Scan(&count).Error; err != nil {
		t.Fatalf("count tenants inside request tx: %v", err)
	}
	if count != 1 {
		t.Fatalf("tenants visible inside request tx = %d, want 1", count)
	}

	var visibleTenant string
	if err := app.WithContext(ctx).Raw(`SELECT id::text FROM tenants`).Scan(&visibleTenant).Error; err != nil {
		t.Fatalf("select tenant inside request tx: %v", err)
	}
	if visibleTenant != alpha.TenantID {
		t.Fatalf("visible tenant = %s, want %s", visibleTenant, alpha.TenantID)
	}

	// Repository helpers receive the request context explicitly; the binding
	// must still apply when the handle was created before the transaction.
	if err := tx.Commit().Error; err != nil {
		t.Fatalf("commit request tx: %v", err)
	}
	committed = true
}

func TestRLSGormBindingRejectsCrossTenantWrites(t *testing.T) {
	db := freshDB(t)
	seedSystem(t, db)
	alpha := createTenant(t, db, "Toko Alpha", "toko-alpha", "owner-alpha@fixphone.test")
	beta := createTenant(t, db, "Toko Beta", "toko-beta", "owner-beta@fixphone.test")

	app := rlsGormApp(t, testutil.SQLDB(t, db))

	// Successful request: writes for the active tenant are allowed and commit.
	ctx, tx := rlsRequestTx(t, app, alpha.TenantID)
	if err := app.WithContext(ctx).Exec(`
		INSERT INTO customers (tenant_id, name, phone, phone_normalized)
		VALUES (?, 'Budi', '08123456789', '628123456789')`, alpha.TenantID).Error; err != nil {
		t.Fatalf("insert own tenant customer through binding: %v", err)
	}
	if err := tx.Commit().Error; err != nil {
		t.Fatalf("commit request tx: %v", err)
	}

	// Forging another tenant_id is rejected by the WITH CHECK policy even
	// though the statement runs through the bound shared handle.
	ctx, tx = rlsRequestTx(t, app, alpha.TenantID)
	err := app.WithContext(ctx).Exec(`
		INSERT INTO customers (tenant_id, name, phone, phone_normalized)
		VALUES (?, 'Curang', '0800000000', '62800000000')`, beta.TenantID).Error
	if err == nil {
		_ = tx.Rollback().Error
		t.Fatal("cross-tenant insert through binding succeeded, want row-level security violation")
	}
	if !strings.Contains(err.Error(), "row-level security") {
		_ = tx.Rollback().Error
		t.Fatalf("expected row-level security violation, got: %v", err)
	}

	// PostgreSQL marks the transaction aborted after the policy violation.
	// middleware.Tenant rolls it back when the handler returns an error, so the
	// rejected row never survives.
	if err := tx.Rollback().Error; err != nil {
		t.Fatalf("rollback request tx: %v", err)
	}

	var count int64
	if err := db.Raw(`SELECT count(*) FROM customers`).Scan(&count).Error; err != nil {
		t.Fatalf("superuser count customers: %v", err)
	}
	if count != 1 {
		t.Fatalf("customers after rejected write = %d, want 1", count)
	}
	if err := db.Raw(`SELECT count(*) FROM customers WHERE tenant_id = ?`, beta.TenantID).Scan(&count).Error; err != nil {
		t.Fatalf("superuser count beta customers: %v", err)
	}
	if count != 0 {
		t.Fatalf("beta customers after rejected write = %d, want 0", count)
	}
}

func TestRLSGormNestedTransactionUsesSavepoint(t *testing.T) {
	db := freshDB(t)
	seedSystem(t, db)
	alpha := createTenant(t, db, "Toko Alpha", "toko-alpha", "owner-alpha@fixphone.test")
	createTenant(t, db, "Toko Beta", "toko-beta", "owner-beta@fixphone.test")

	app := rlsGormApp(t, testutil.SQLDB(t, db))

	ctx, tx := rlsRequestTx(t, app, alpha.TenantID)
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback().Error
		}
	}()

	if err := app.WithContext(ctx).Exec(`
		INSERT INTO customers (tenant_id, name, phone, phone_normalized)
		VALUES (?, 'Budi', '08123456789', '628123456789')`, alpha.TenantID).Error; err != nil {
		t.Fatalf("insert customer in request tx: %v", err)
	}

	// A service-level transaction started with database.WithContext must use a
	// savepoint on the request transaction, otherwise it would run on a fresh
	// pooled connection without app.tenant_id and fail closed.
	savepointErr := database.WithContext(ctx, app).Transaction(func(inner *gorm.DB) error {
		if err := inner.Exec(`
			INSERT INTO customers (tenant_id, name, phone, phone_normalized)
			VALUES (?, 'Sementara', '0811111111', '62811111111')`, alpha.TenantID).Error; err != nil {
			return fmt.Errorf("insert inside savepoint: %w", err)
		}
		var inside int64
		if err := inner.Raw(`SELECT count(*) FROM customers`).Scan(&inside).Error; err != nil {
			return fmt.Errorf("count inside savepoint: %w", err)
		}
		if inside != 2 {
			return fmt.Errorf("customers inside savepoint = %d, want 2", inside)
		}
		return errors.New("batalkan savepoint")
	})
	if savepointErr == nil || savepointErr.Error() != "batalkan savepoint" {
		t.Fatalf("nested transaction error = %v, want rollback request", savepointErr)
	}

	// Rolling back the savepoint keeps the outer transaction usable.
	var count int64
	if err := app.WithContext(ctx).Raw(`SELECT count(*) FROM customers`).Scan(&count).Error; err != nil {
		t.Fatalf("count customers after savepoint rollback: %v", err)
	}
	if count != 1 {
		t.Fatalf("customers after savepoint rollback = %d, want 1", count)
	}

	// A successful nested transaction is released with the outer commit.
	if err := database.WithContext(ctx, app).Transaction(func(inner *gorm.DB) error {
		return inner.Exec(`
			INSERT INTO customers (tenant_id, name, phone, phone_normalized)
			VALUES (?, 'Siti', '0822222222', '62822222222')`, alpha.TenantID).Error
	}); err != nil {
		t.Fatalf("commit nested transaction: %v", err)
	}

	if err := tx.Commit().Error; err != nil {
		t.Fatalf("commit request tx: %v", err)
	}
	committed = true

	if err := db.Raw(`SELECT count(*) FROM customers`).Scan(&count).Error; err != nil {
		t.Fatalf("superuser count customers: %v", err)
	}
	if count != 2 {
		t.Fatalf("customers after commit = %d, want 2", count)
	}
}
