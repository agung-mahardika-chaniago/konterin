package integration_test

import (
	"context"
	"strings"
	"testing"

	"fixphone/backend/internal/database"

	"gorm.io/gorm"
)

const (
	testSuperadminEmail    = "superadmin@fixphone.test"
	testSuperadminPassword = "Superadmin123"
	testOwnerPassword      = "Owner12345"
)

type tenantFixture struct {
	TenantID string
	OutletID string
	OwnerID  string
}

func seedSystem(t *testing.T, db *gorm.DB) *database.SeedResult {
	t.Helper()
	result, err := database.Seed(context.Background(), db, database.SeedOptions{
		SuperadminName:     "Super Admin Test",
		SuperadminEmail:    testSuperadminEmail,
		SuperadminPassword: testSuperadminPassword,
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	return result
}

func createTenant(t *testing.T, db *gorm.DB, name, slug, email string) tenantFixture {
	t.Helper()
	result, err := database.CreateTenant(context.Background(), db, database.CreateTenantInput{
		Name:          name,
		Slug:          slug,
		OwnerName:     "Owner " + name,
		OwnerEmail:    email,
		OwnerPassword: testOwnerPassword,
		OutletName:    "Outlet " + name,
		OutletCode:    "MAIN",
		Timezone:      "Asia/Jakarta",
		Phone:         "0211234567",
		Address:       "Jl. Contoh No. 1",
	})
	if err != nil {
		t.Fatalf("create tenant %s: %v", slug, err)
	}
	return tenantFixture{
		TenantID: result.TenantID.String(),
		OutletID: result.OutletID.String(),
		OwnerID:  result.OwnerID.String(),
	}
}

func assertConstraintViolation(t *testing.T, err error, constraint string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected constraint %s to reject the row, but insert succeeded", constraint)
	}
	if !strings.Contains(err.Error(), constraint) {
		t.Fatalf("expected constraint %s, got error: %v", constraint, err)
	}
}

func TestSeedIsIdempotent(t *testing.T) {
	db := freshDB(t)

	first := seedSystem(t, db)
	if first.Roles != 5 {
		t.Fatalf("roles = %d, want 5", first.Roles)
	}
	if first.Permissions != 48 {
		t.Fatalf("permissions = %d, want 48", first.Permissions)
	}
	if first.RolePermissions != 123 {
		t.Fatalf("role_permissions = %d, want 123", first.RolePermissions)
	}
	if first.Superadmins != 1 {
		t.Fatalf("superadmins = %d, want 1", first.Superadmins)
	}

	ownerGrants := permissionCodesForRole(t, db, "owner")
	if len(ownerGrants) != 46 {
		t.Fatalf("owner permissions = %d, want 46", len(ownerGrants))
	}
	for _, forbidden := range []string{"platform.tenant.view", "platform.tenant.manage"} {
		for _, code := range ownerGrants {
			if code == forbidden {
				t.Fatalf("owner must not have platform permission %s", forbidden)
			}
		}
	}

	adminGrants := permissionSet(permissionCodesForRole(t, db, "admin"))
	for _, forbidden := range []string{
		"service.credential.reveal",
		"service.override_unpaid_pickup",
		"report.profit.view",
		"audit.view",
	} {
		if adminGrants[forbidden] {
			t.Fatalf("admin must not have %s", forbidden)
		}
	}

	// Creating a tenant after the first seed proves category seeding is also
	// repeatable, because the second seed backfills categories per tenant.
	createTenant(t, db, "Toko Seed", "toko-seed", "owner.seed@example.com")

	second := seedSystem(t, db)
	if second.Roles != first.Roles ||
		second.Permissions != first.Permissions ||
		second.RolePermissions != first.RolePermissions ||
		second.Superadmins != first.Superadmins {
		t.Fatalf("seed is not idempotent: first=%+v second=%+v", first, second)
	}
	if second.ExpenseCategories != 6 {
		t.Fatalf("expense categories = %d, want 6", second.ExpenseCategories)
	}
	if testutilCount(t, db, "expense_categories") != 6 {
		t.Fatalf("expense_categories rows duplicated")
	}

	t.Run("superadmin email cannot hijack a tenant user", func(t *testing.T) {
		_, err := database.Seed(context.Background(), db, database.SeedOptions{
			SuperadminName:     "Super Admin Test",
			SuperadminEmail:    "owner.seed@example.com",
			SuperadminPassword: testSuperadminPassword,
		})
		if err == nil || !strings.Contains(err.Error(), "pengguna tenant") {
			t.Fatalf("expected tenant email error, got %v", err)
		}
	})
}

func TestCreateTenantProvisioning(t *testing.T) {
	db := freshDB(t)
	seedSystem(t, db)

	tenant := createTenant(t, db, "Toko Makmur", "toko-makmur", "owner@toko-makmur.test")

	var ownerRole, tenantName, outletCode, settingsStore string
	if err := db.Raw(`SELECT r.slug FROM users u JOIN roles r ON r.id = u.role_id WHERE u.id = ?`, tenant.OwnerID).Scan(&ownerRole).Error; err != nil {
		t.Fatalf("read owner role: %v", err)
	}
	if ownerRole != "owner" {
		t.Fatalf("owner role = %q, want owner", ownerRole)
	}
	if err := db.Raw(`SELECT name FROM tenants WHERE id = ?`, tenant.TenantID).Scan(&tenantName).Error; err != nil {
		t.Fatalf("read tenant: %v", err)
	}
	if tenantName != "Toko Makmur" {
		t.Fatalf("tenant name = %q", tenantName)
	}
	if err := db.Raw(`SELECT code FROM outlets WHERE id = ?`, tenant.OutletID).Scan(&outletCode).Error; err != nil {
		t.Fatalf("read outlet: %v", err)
	}
	if outletCode != "MAIN" {
		t.Fatalf("outlet code = %q, want MAIN", outletCode)
	}
	if err := db.Raw(`SELECT store_name FROM business_settings WHERE tenant_id = ?`, tenant.TenantID).Scan(&settingsStore).Error; err != nil {
		t.Fatalf("read settings: %v", err)
	}
	if settingsStore != "Toko Makmur" {
		t.Fatalf("settings store_name = %q", settingsStore)
	}
	if got := testutilCount(t, db, "expense_categories"); got != 6 {
		t.Fatalf("expense categories = %d, want 6", got)
	}

	t.Run("duplicate slug is rejected", func(t *testing.T) {
		_, err := database.CreateTenant(context.Background(), db, database.CreateTenantInput{
			Name:          "Toko Lain",
			Slug:          "toko-makmur",
			OwnerName:     "Owner Lain",
			OwnerEmail:    "owner.lain@example.com",
			OwnerPassword: testOwnerPassword,
			OutletName:    "Outlet Lain",
			OutletCode:    "MAIN",
		})
		if err == nil || !strings.Contains(err.Error(), "sudah ada") {
			t.Fatalf("expected duplicate slug error, got %v", err)
		}
	})

	t.Run("duplicate owner email is rejected", func(t *testing.T) {
		_, err := database.CreateTenant(context.Background(), db, database.CreateTenantInput{
			Name:          "Toko Lain",
			Slug:          "toko-lain",
			OwnerName:     "Owner Lain",
			OwnerEmail:    "owner@toko-makmur.test",
			OwnerPassword: testOwnerPassword,
			OutletName:    "Outlet Lain",
			OutletCode:    "MAIN",
		})
		if err == nil || !strings.Contains(err.Error(), "sudah dipakai") {
			t.Fatalf("expected duplicate email error, got %v", err)
		}
	})

	t.Run("weak owner password is rejected", func(t *testing.T) {
		_, err := database.CreateTenant(context.Background(), db, database.CreateTenantInput{
			Name:          "Toko Lain",
			Slug:          "toko-lain",
			OwnerName:     "Owner Lain",
			OwnerEmail:    "owner.lain@example.com",
			OwnerPassword: "short",
			OutletName:    "Outlet Lain",
			OutletCode:    "MAIN",
		})
		if err == nil {
			t.Fatal("expected weak password to be rejected")
		}
	})
}

func permissionCodesForRole(t *testing.T, db *gorm.DB, roleSlug string) []string {
	t.Helper()
	var codes []string
	if err := db.Raw(`SELECT p.code
		FROM permissions p
		JOIN role_permissions rp ON rp.permission_id = p.id
		JOIN roles r ON r.id = rp.role_id
		WHERE r.slug = ?
		ORDER BY p.code`, roleSlug).Scan(&codes).Error; err != nil {
		t.Fatalf("read permissions for %s: %v", roleSlug, err)
	}
	return codes
}

func permissionSet(codes []string) map[string]bool {
	set := make(map[string]bool, len(codes))
	for _, code := range codes {
		set[code] = true
	}
	return set
}

func testutilCount(t *testing.T, db *gorm.DB, table string) int64 {
	t.Helper()
	var count int64
	if err := db.Raw(`SELECT count(*) FROM ` + table).Scan(&count).Error; err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return count
}
