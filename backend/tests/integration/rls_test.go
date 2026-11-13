package integration_test

import (
	"context"
	"database/sql"
	"net/url"
	"strings"
	"testing"
	"time"

	"fixphone/backend/tests/testutil"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// The RLS isolation tests connect as the non-superuser application role.
// PostgreSQL lets table owners and superusers bypass row level security, so
// using the default (superuser) test connection would prove nothing.
const (
	rlsAppRole     = "fixphone_app"
	rlsAppPassword = "rls-test-password"
)

// rlsOpenApp grants the application role a login on the disposable test
// database and opens a dedicated connection as that role.
func rlsOpenApp(t *testing.T, admin *sql.DB) *sql.DB {
	t.Helper()

	if _, err := admin.Exec(`ALTER ROLE fixphone_app WITH LOGIN PASSWORD '` + rlsAppPassword + `'`); err != nil {
		t.Fatalf("enable login for %s: %v", rlsAppRole, err)
	}

	u, err := url.Parse(testutil.DSN())
	if err != nil {
		t.Fatalf("parse test DSN: %v", err)
	}
	u.User = url.UserPassword(rlsAppRole, rlsAppPassword)

	db, err := sql.Open("pgx", u.String())
	if err != nil {
		t.Fatalf("open app role connection: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		t.Fatalf("ping as %s: %v", rlsAppRole, err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// rlsTenantTx runs fn inside a transaction scoped to tenantID. An empty
// tenantID leaves app.tenant_id unset, which every policy must treat as "no
// access". SET LOCAL keeps the setting from leaking through the connection
// pool when the transaction ends.
func rlsTenantTx(t *testing.T, db *sql.DB, tenantID string, fn func(tx *sql.Tx)) {
	t.Helper()

	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	if tenantID != "" {
		if _, err := tx.Exec(`SELECT set_config('app.tenant_id', $1, true)`, tenantID); err != nil {
			t.Fatalf("set app.tenant_id: %v", err)
		}
	}
	fn(tx)
}

// rlsTenantTxCommit is rlsTenantTx with a commit, for statements whose effect
// must survive the transaction (e.g. proving the app role may append audit
// logs).
func rlsTenantTxCommit(t *testing.T, db *sql.DB, tenantID string, fn func(tx *sql.Tx)) {
	t.Helper()

	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	if tenantID != "" {
		if _, err := tx.Exec(`SELECT set_config('app.tenant_id', $1, true)`, tenantID); err != nil {
			_ = tx.Rollback()
			t.Fatalf("set app.tenant_id: %v", err)
		}
	}
	fn(tx)
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit tx: %v", err)
	}
}

func rlsCount(t *testing.T, tx *sql.Tx, query string, args ...any) int {
	t.Helper()
	var n int
	if err := tx.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return n
}

func rlsScalar(t *testing.T, tx *sql.Tx, query string, args ...any) string {
	t.Helper()
	var v string
	if err := tx.QueryRow(query, args...).Scan(&v); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return v
}

func rlsExpectError(t *testing.T, tx *sql.Tx, query string, args ...any) string {
	t.Helper()
	if _, err := tx.Exec(query, args...); err == nil {
		t.Fatalf("expected an error from %q", query)
	} else {
		return err.Error()
	}
	return ""
}

func TestRLSRequiresTenantContext(t *testing.T) {
	db := freshDB(t)
	seedSystem(t, db)
	createTenant(t, db, "Toko Alpha", "toko-alpha", "owner-alpha@fixphone.test")
	createTenant(t, db, "Toko Beta", "toko-beta", "owner-beta@fixphone.test")

	admin := testutil.SQLDB(t, db)
	if got := testutil.Count(t, db, "tenants"); got != 2 {
		t.Fatalf("superuser should see both tenants, got %d", got)
	}
	app := rlsOpenApp(t, admin)

	// No app.tenant_id at all: fail closed.
	rlsTenantTx(t, app, "", func(tx *sql.Tx) {
		for _, table := range []string{"tenants", "outlets", "users", "business_settings", "customers"} {
			if n := rlsCount(t, tx, `SELECT count(*) FROM `+table); n != 0 {
				t.Errorf("%s leaked %d rows without a tenant context", table, n)
			}
		}
	})

	// Empty string must behave exactly like "unset".
	rlsTenantTx(t, app, "", func(tx *sql.Tx) {
		if _, err := tx.Exec(`SELECT set_config('app.tenant_id', '', true)`); err != nil {
			t.Fatalf("set empty app.tenant_id: %v", err)
		}
		if n := rlsCount(t, tx, `SELECT count(*) FROM tenants`); n != 0 {
			t.Errorf("tenants leaked %d rows with an empty tenant context", n)
		}
	})
}

func TestRLSScopesRowsToActiveTenant(t *testing.T) {
	db := freshDB(t)
	seedSystem(t, db)
	alpha := createTenant(t, db, "Toko Alpha", "toko-alpha", "owner-alpha@fixphone.test")
	beta := createTenant(t, db, "Toko Beta", "toko-beta", "owner-beta@fixphone.test")

	app := rlsOpenApp(t, testutil.SQLDB(t, db))

	cases := []struct {
		name     string
		tenantID string
		outletID string
		ownerID  string
	}{
		{"alpha", alpha.TenantID, alpha.OutletID, alpha.OwnerID},
		{"beta", beta.TenantID, beta.OutletID, beta.OwnerID},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rlsTenantTx(t, app, tc.tenantID, func(tx *sql.Tx) {
				if n := rlsCount(t, tx, `SELECT count(*) FROM tenants`); n != 1 {
					t.Errorf("tenants = %d, want 1", n)
				}
				if got := rlsScalar(t, tx, `SELECT id::text FROM tenants`); got != tc.tenantID {
					t.Errorf("visible tenant = %s, want %s", got, tc.tenantID)
				}
				if n := rlsCount(t, tx, `SELECT count(*) FROM outlets`); n != 1 {
					t.Errorf("outlets = %d, want 1", n)
				}
				if got := rlsScalar(t, tx, `SELECT id::text FROM outlets`); got != tc.outletID {
					t.Errorf("visible outlet = %s, want %s", got, tc.outletID)
				}
				// The superadmin (tenant_id IS NULL) must never be visible to a
				// tenant-scoped connection.
				if n := rlsCount(t, tx, `SELECT count(*) FROM users`); n != 1 {
					t.Errorf("users = %d, want 1", n)
				}
				if got := rlsScalar(t, tx, `SELECT id::text FROM users`); got != tc.ownerID {
					t.Errorf("visible user = %s, want %s", got, tc.ownerID)
				}
				if n := rlsCount(t, tx, `SELECT count(*) FROM business_settings`); n != 1 {
					t.Errorf("business_settings = %d, want 1", n)
				}
				if n := rlsCount(t, tx, `SELECT count(*) FROM expense_categories`); n != 6 {
					t.Errorf("expense_categories = %d, want 6", n)
				}
			})
		})
	}
}

func TestRLSBlocksCrossTenantWrites(t *testing.T) {
	db := freshDB(t)
	seedSystem(t, db)
	alpha := createTenant(t, db, "Toko Alpha", "toko-alpha", "owner-alpha@fixphone.test")
	beta := createTenant(t, db, "Toko Beta", "toko-beta", "owner-beta@fixphone.test")

	app := rlsOpenApp(t, testutil.SQLDB(t, db))

	// A row for the active tenant is accepted.
	rlsTenantTx(t, app, alpha.TenantID, func(tx *sql.Tx) {
		if _, err := tx.Exec(`
			INSERT INTO customers (tenant_id, name, phone, phone_normalized)
			VALUES ($1, 'Budi', '08123456789', '628123456789')`, alpha.TenantID); err != nil {
			t.Fatalf("insert own tenant customer: %v", err)
		}
	})

	// A row claiming another tenant is rejected by WITH CHECK.
	rlsTenantTx(t, app, alpha.TenantID, func(tx *sql.Tx) {
		msg := rlsExpectError(t, tx, `
			INSERT INTO customers (tenant_id, name, phone, phone_normalized)
			VALUES ($1, 'Curang', '0800000000', '62800000000')`, beta.TenantID)
		if !strings.Contains(msg, "row-level security") {
			t.Fatalf("expected row-level security violation, got: %s", msg)
		}
	})

	// Foreign rows are invisible, so UPDATE/DELETE affect nothing.
	rlsTenantTx(t, app, alpha.TenantID, func(tx *sql.Tx) {
		res, err := tx.Exec(`UPDATE outlets SET name = 'diretas' WHERE id = $1`, beta.OutletID)
		if err != nil {
			t.Fatalf("update foreign outlet: %v", err)
		}
		if n, _ := res.RowsAffected(); n != 0 {
			t.Errorf("update touched %d foreign rows, want 0", n)
		}
	})
	rlsTenantTx(t, app, alpha.TenantID, func(tx *sql.Tx) {
		res, err := tx.Exec(`DELETE FROM outlets WHERE id = $1`, beta.OutletID)
		if err != nil {
			t.Fatalf("delete foreign outlet: %v", err)
		}
		if n, _ := res.RowsAffected(); n != 0 {
			t.Errorf("delete removed %d foreign rows, want 0", n)
		}
	})

	// The beta row must still be intact when checked as superuser.
	var name string
	if err := db.Raw(`SELECT name FROM outlets WHERE id = ?`, beta.OutletID).Scan(&name).Error; err != nil {
		t.Fatalf("read beta outlet: %v", err)
	}
	if name != "Outlet Toko Beta" {
		t.Fatalf("beta outlet was modified, name = %q", name)
	}
}

func TestRLSProtectsIndirectTenantTables(t *testing.T) {
	db := freshDB(t)
	seedSystem(t, db)
	alpha := createTenant(t, db, "Toko Alpha", "toko-alpha", "owner-alpha@fixphone.test")
	beta := createTenant(t, db, "Toko Beta", "toko-beta", "owner-beta@fixphone.test")

	// Seed indirect rows for alpha only.
	if err := db.Exec(`INSERT INTO user_outlets (user_id, outlet_id) VALUES (?, ?)`,
		alpha.OwnerID, alpha.OutletID).Error; err != nil {
		t.Fatalf("seed user_outlets: %v", err)
	}
	if err := db.Exec(`
		INSERT INTO refresh_tokens (user_id, token_hash, family_id, expires_at)
		VALUES (?, repeat('a', 64), gen_random_uuid(), now() + interval '1 day')`,
		alpha.OwnerID).Error; err != nil {
		t.Fatalf("seed refresh_tokens: %v", err)
	}
	if err := db.Exec(`
		INSERT INTO notifications (tenant_id, user_id, type, title)
		VALUES (?, ?, 'system', 'Halo Alpha')`, alpha.TenantID, alpha.OwnerID).Error; err != nil {
		t.Fatalf("seed notifications: %v", err)
	}
	if err := db.Exec(`
		INSERT INTO notification_reads (notification_id, user_id)
		SELECT id, ? FROM notifications WHERE tenant_id = ? LIMIT 1`,
		alpha.OwnerID, alpha.TenantID).Error; err != nil {
		t.Fatalf("seed notification_reads: %v", err)
	}

	app := rlsOpenApp(t, testutil.SQLDB(t, db))

	for _, tc := range []struct {
		name     string
		tenantID string
		want     int
	}{
		{"alpha", alpha.TenantID, 1},
		{"beta", beta.TenantID, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rlsTenantTx(t, app, tc.tenantID, func(tx *sql.Tx) {
				for _, table := range []string{"refresh_tokens", "user_outlets", "notification_reads"} {
					if n := rlsCount(t, tx, `SELECT count(*) FROM `+table); n != tc.want {
						t.Errorf("%s = %d, want %d", table, n, tc.want)
					}
				}
			})
		})
	}
}

func TestRLSAuditLogIsAppendOnly(t *testing.T) {
	db := freshDB(t)
	seedSystem(t, db)
	alpha := createTenant(t, db, "Toko Alpha", "toko-alpha", "owner-alpha@fixphone.test")
	beta := createTenant(t, db, "Toko Beta", "toko-beta", "owner-beta@fixphone.test")

	app := rlsOpenApp(t, testutil.SQLDB(t, db))

	rlsTenantTxCommit(t, app, alpha.TenantID, func(tx *sql.Tx) {
		if _, err := tx.Exec(`
			INSERT INTO audit_logs (tenant_id, action, module, entity_type)
			VALUES ($1, 'login.success', 'auth', 'user')`, alpha.TenantID); err != nil {
			t.Fatalf("append audit log: %v", err)
		}
	})

	rlsTenantTx(t, app, alpha.TenantID, func(tx *sql.Tx) {
		if n := rlsCount(t, tx, `SELECT count(*) FROM audit_logs`); n != 1 {
			t.Errorf("audit_logs visible = %d, want 1", n)
		}
	})
	rlsTenantTx(t, app, beta.TenantID, func(tx *sql.Tx) {
		if n := rlsCount(t, tx, `SELECT count(*) FROM audit_logs`); n != 0 {
			t.Errorf("audit_logs leaked %d rows to another tenant", n)
		}
	})

	rlsTenantTx(t, app, alpha.TenantID, func(tx *sql.Tx) {
		msg := rlsExpectError(t, tx, `UPDATE audit_logs SET action = 'tampered'`)
		if !strings.Contains(msg, "permission denied") {
			t.Fatalf("expected permission denied on update, got: %s", msg)
		}
	})
	rlsTenantTx(t, app, alpha.TenantID, func(tx *sql.Tx) {
		msg := rlsExpectError(t, tx, `DELETE FROM audit_logs`)
		if !strings.Contains(msg, "permission denied") {
			t.Fatalf("expected permission denied on delete, got: %s", msg)
		}
	})
}
