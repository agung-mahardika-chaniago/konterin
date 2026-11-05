package database

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// BcryptCost is the project-wide password hashing cost.
const BcryptCost = 12

type systemRole struct {
	Slug string
	Name string
}

var systemRoles = []systemRole{
	{Slug: "super_admin", Name: "Super Admin"},
	{Slug: "owner", Name: "Owner"},
	{Slug: "admin", Name: "Admin"},
	{Slug: "technician", Name: "Teknisi"},
	{Slug: "cashier", Name: "Kasir"},
}

type systemPermission struct {
	Code        string
	Module      string
	Description string
}

// systemPermissions is the single source of truth for the RBAC seed.
var systemPermissions = []systemPermission{
	{"dashboard.view", "dashboard", "Melihat dashboard"},

	{"service.view", "service", "Melihat data servis"},
	{"service.create", "service", "Menerima servis baru"},
	{"service.update", "service", "Mengubah data servis"},
	{"service.status", "service", "Mengubah status servis"},
	{"service.assign", "service", "Menugaskan teknisi"},
	{"service.cancel", "service", "Membatalkan servis"},
	{"service.credential.reveal", "service", "Melihat kredensial perangkat"},
	{"service.override_unpaid_pickup", "service", "Mengizinkan pengambilan sebelum lunas"},

	{"diagnosis.view", "diagnosis", "Melihat diagnosis"},
	{"diagnosis.manage", "diagnosis", "Mengelola diagnosis"},

	{"customer.view", "customer", "Melihat pelanggan"},
	{"customer.create", "customer", "Menambah pelanggan"},
	{"customer.update", "customer", "Mengubah pelanggan"},
	{"customer.delete", "customer", "Menghapus pelanggan"},

	{"sparepart.view", "sparepart", "Melihat sparepart"},
	{"sparepart.create", "sparepart", "Menambah sparepart"},
	{"sparepart.update", "sparepart", "Mengubah sparepart"},
	{"sparepart.delete", "sparepart", "Menghapus sparepart"},

	{"inventory.view", "inventory", "Melihat mutasi stok"},
	{"inventory.purchase", "inventory", "Mencatat pembelian stok"},
	{"inventory.adjust", "inventory", "Melakukan opname stok"},
	{"inventory.cost.view", "inventory", "Melihat harga modal"},

	{"invoice.view", "invoice", "Melihat invoice"},
	{"invoice.create", "invoice", "Membuat invoice"},
	{"invoice.update", "invoice", "Mengubah invoice"},
	{"invoice.void", "invoice", "Membatalkan invoice"},

	{"payment.view", "payment", "Melihat pembayaran"},
	{"payment.create", "payment", "Menerima pembayaran"},
	{"payment.void", "payment", "Membatalkan pembayaran"},

	{"warranty.view", "warranty", "Melihat garansi"},
	{"warranty.claim", "warranty", "Mengelola klaim garansi"},

	{"report.view", "report", "Melihat laporan"},
	{"report.export", "report", "Mengekspor laporan"},
	{"report.profit.view", "report", "Melihat laporan laba"},

	{"expense.view", "expense", "Melihat pengeluaran"},
	{"expense.create", "expense", "Menambah pengeluaran"},
	{"expense.update", "expense", "Mengubah pengeluaran"},
	{"expense.delete", "expense", "Menghapus pengeluaran"},

	{"user.view", "user", "Melihat pengguna"},
	{"user.manage", "user", "Mengelola pengguna"},

	{"outlet.view", "outlet", "Melihat outlet"},
	{"outlet.manage", "outlet", "Mengelola outlet"},

	{"settings.view", "settings", "Melihat pengaturan"},
	{"settings.manage", "settings", "Mengubah pengaturan"},

	{"audit.view", "audit", "Melihat audit log"},

	{"platform.tenant.view", "platform", "Melihat tenant platform"},
	{"platform.tenant.manage", "platform", "Mengelola tenant platform"},
}

// tenantPermissions is every permission except the platform module.
func tenantPermissions() []string {
	out := make([]string, 0, len(systemPermissions)-2)
	for _, p := range systemPermissions {
		if p.Module != "platform" {
			out = append(out, p.Code)
		}
	}
	return out
}

func withoutPermissions(all []string, excluded ...string) []string {
	skip := make(map[string]struct{}, len(excluded))
	for _, code := range excluded {
		skip[code] = struct{}{}
	}
	out := make([]string, 0, len(all))
	for _, code := range all {
		if _, found := skip[code]; !found {
			out = append(out, code)
		}
	}
	return out
}

func rolePermissionMap() map[string][]string {
	all := tenantPermissions()
	return map[string][]string{
		"owner": all,
		"admin": withoutPermissions(all,
			"service.credential.reveal",
			"service.override_unpaid_pickup",
			"report.profit.view",
			"audit.view",
		),
		"technician": {
			"dashboard.view",
			"service.view", "service.create", "service.update", "service.status",
			"service.credential.reveal",
			"diagnosis.view", "diagnosis.manage",
			"customer.view",
			"sparepart.view", "inventory.view",
			"warranty.view", "warranty.claim",
			"outlet.view",
		},
		"cashier": {
			"dashboard.view",
			"service.view", "service.create", "service.update", "service.status",
			"customer.view", "customer.create", "customer.update",
			"sparepart.view", "inventory.view",
			"invoice.view", "invoice.create", "invoice.update",
			"payment.view", "payment.create",
			"warranty.view", "warranty.claim",
			"report.view",
			"outlet.view",
		},
		"super_admin": {
			"platform.tenant.view", "platform.tenant.manage",
		},
	}
}

// DefaultExpenseCategories are created for every tenant.
var DefaultExpenseCategories = []string{
	"Operasional",
	"Gaji",
	"Sewa",
	"Listrik & Internet",
	"Pembelian Alat",
	"Lain-lain",
}

// SeedOptions controls the idempotent system seed.
type SeedOptions struct {
	SuperadminName     string
	SuperadminEmail    string
	SuperadminPassword string
}

// SeedResult reports how many rows exist after the seed.
type SeedResult struct {
	Roles             int64
	Permissions       int64
	RolePermissions   int64
	Superadmins       int64
	ExpenseCategories int64
}

// Seed inserts/updates system roles, permissions, their mapping, the super
// admin account, and default expense categories. Running it repeatedly is safe.
func Seed(ctx context.Context, db *gorm.DB, opts SeedOptions) (*SeedResult, error) {
	email, err := NormalizeEmail(opts.SuperadminEmail)
	if err != nil {
		return nil, fmt.Errorf("SEED_SUPERADMIN_EMAIL: %w", err)
	}
	if err := ValidatePassword(opts.SuperadminPassword); err != nil {
		return nil, fmt.Errorf("SEED_SUPERADMIN_PASSWORD: %w", err)
	}
	name := strings.TrimSpace(opts.SuperadminName)
	if name == "" {
		name = "Super Admin"
	}
	if len(name) > 120 {
		return nil, errors.New("nama super admin maksimal 120 karakter")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(opts.SuperadminPassword), BcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash password super admin: %w", err)
	}

	result := &SeedResult{}
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		roleIDs := make(map[string]uuid.UUID, len(systemRoles))
		for _, role := range systemRoles {
			id, err := insertReturningID(tx, `
				INSERT INTO roles (slug, name, is_system)
				VALUES (?, ?, true)
				ON CONFLICT (slug) DO UPDATE
				SET name = EXCLUDED.name, is_system = true
				RETURNING id`, role.Slug, role.Name)
			if err != nil {
				return fmt.Errorf("seed role %s: %w", role.Slug, err)
			}
			roleIDs[role.Slug] = id
		}

		permissionIDs := make(map[string]uuid.UUID, len(systemPermissions))
		for _, permission := range systemPermissions {
			id, err := insertReturningID(tx, `
				INSERT INTO permissions (code, module, description)
				VALUES (?, ?, ?)
				ON CONFLICT (code) DO UPDATE
				SET module = EXCLUDED.module, description = EXCLUDED.description
				RETURNING id`,
				permission.Code, permission.Module, permission.Description)
			if err != nil {
				return fmt.Errorf("seed permission %s: %w", permission.Code, err)
			}
			permissionIDs[permission.Code] = id
		}

		// System role mappings are rebuilt to keep them deterministic. Custom
		// roles are never touched.
		systemRoleIDs := make([]uuid.UUID, 0, len(roleIDs))
		for _, id := range roleIDs {
			systemRoleIDs = append(systemRoleIDs, id)
		}
		if err := tx.Exec(`DELETE FROM role_permissions WHERE role_id IN ?`, systemRoleIDs).Error; err != nil {
			return fmt.Errorf("membersihkan mapping role: %w", err)
		}

		for roleSlug, codes := range rolePermissionMap() {
			roleID := roleIDs[roleSlug]
			for _, code := range codes {
				permissionID, ok := permissionIDs[code]
				if !ok {
					return fmt.Errorf("permission tidak dikenal pada mapping %s: %s", roleSlug, code)
				}
				if err := tx.Exec(`
					INSERT INTO role_permissions (role_id, permission_id)
					VALUES (?, ?) ON CONFLICT DO NOTHING`, roleID, permissionID).Error; err != nil {
					return fmt.Errorf("seed mapping %s/%s: %w", roleSlug, code, err)
				}
			}
		}

		superRoleID := roleIDs["super_admin"]
		var existing struct {
			ID        string `gorm:"column:id"`
			HasTenant bool   `gorm:"column:has_tenant"`
		}
		if err := tx.Raw(`SELECT id::text AS id, (tenant_id IS NOT NULL) AS has_tenant
			FROM users WHERE email = ? AND deleted_at IS NULL`, email).
			Scan(&existing).Error; err != nil {
			return fmt.Errorf("mencari super admin: %w", err)
		}
		if existing.HasTenant {
			return fmt.Errorf("email %q sudah dipakai pengguna tenant; gunakan email super admin lain", email)
		}
		if existing.ID == "" {
			if err := tx.Exec(`
				INSERT INTO users (tenant_id, role_id, name, email, password_hash, is_active, must_change_password)
				VALUES (NULL, ?, ?, ?, ?, true, true)`,
				superRoleID, name, email, string(hash)).Error; err != nil {
				return fmt.Errorf("membuat super admin: %w", err)
			}
		} else {
			userID, err := uuid.Parse(existing.ID)
			if err != nil {
				return fmt.Errorf("membaca ID super admin: %w", err)
			}
			if err := tx.Exec(`
				UPDATE users
				SET role_id = ?, name = ?, is_active = true, deleted_at = NULL, updated_at = now()
				WHERE id = ?`, superRoleID, name, userID).Error; err != nil {
				return fmt.Errorf("memperbarui super admin: %w", err)
			}
		}

		if err := tx.Exec(`
			INSERT INTO expense_categories (tenant_id, name, is_system)
			SELECT t.id, v.name, true
			FROM tenants t
			CROSS JOIN (VALUES
				('Operasional'), ('Gaji'), ('Sewa'),
				('Listrik & Internet'), ('Pembelian Alat'), ('Lain-lain')
			) AS v(name)
			ON CONFLICT (tenant_id, name) DO NOTHING`).Error; err != nil {
			return fmt.Errorf("seed kategori pengeluaran: %w", err)
		}

		counts := []struct {
			query string
			dest  *int64
		}{
			{`SELECT count(*) FROM roles`, &result.Roles},
			{`SELECT count(*) FROM permissions`, &result.Permissions},
			{`SELECT count(*) FROM role_permissions`, &result.RolePermissions},
			{`SELECT count(*) FROM users WHERE email = ? AND deleted_at IS NULL`, &result.Superadmins},
			{`SELECT count(*) FROM expense_categories`, &result.ExpenseCategories},
		}
		for _, item := range counts {
			var count int64
			args := []any{}
			if strings.Contains(item.query, "email = ?") {
				args = append(args, email)
			}
			if err := tx.Raw(item.query, args...).Scan(&count).Error; err != nil {
				return fmt.Errorf("menghitung hasil seed: %w", err)
			}
			*item.dest = count
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// insertReturningID executes an INSERT ... RETURNING id statement and parses
// the UUID as text. pgx returns UUID columns as strings, which GORM cannot scan
// directly into uuid.UUID.
func insertReturningID(tx *gorm.DB, query string, args ...any) (uuid.UUID, error) {
	var raw string
	if err := tx.Raw(query, args...).Scan(&raw).Error; err != nil {
		return uuid.Nil, err
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse UUID %q: %w", raw, err)
	}
	return id, nil
}
