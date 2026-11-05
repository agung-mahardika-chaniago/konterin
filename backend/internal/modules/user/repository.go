package user

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// ErrNotFound is returned when a tenant-scoped record does not exist.
var ErrNotFound = errors.New("data tidak ditemukan")

// Repository is the tenant-scoped data access layer for identity records.
type Repository struct {
	db *gorm.DB
}

// NewRepository builds a Repository.
func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

const userSelect = `
	SELECT u.id, u.tenant_id, u.role_id, r.slug AS role_slug, r.name AS role_name,
	       u.name, u.email, u.phone, u.password_hash, u.avatar_path, u.is_active,
	       u.failed_login_count, u.locked_until, u.last_login_at,
	       u.permission_version, u.must_change_password, u.created_at, u.updated_at
	FROM users u
	JOIN roles r ON r.id = u.role_id`

// FindByEmail loads an active (not soft-deleted) user by normalized email.
func (r *Repository) FindByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := r.db.WithContext(ctx).Raw(userSelect+`
		WHERE u.email = ? AND u.deleted_at IS NULL`, email).Scan(&u).Error
	if err != nil {
		return nil, err
	}
	if u.ID.String() == "00000000-0000-0000-0000-000000000000" {
		return nil, ErrNotFound
	}
	return &u, nil
}

// FindByID loads a user by ID regardless of soft-delete state.
func (r *Repository) FindByID(ctx context.Context, id interface{}) (*User, error) {
	var u User
	err := r.db.WithContext(ctx).Raw(userSelect+`
		WHERE u.id = ? AND u.deleted_at IS NULL`, id).Scan(&u).Error
	if err != nil {
		return nil, err
	}
	if u.ID.String() == "00000000-0000-0000-0000-000000000000" {
		return nil, ErrNotFound
	}
	return &u, nil
}

// PermissionCodes returns every permission code granted to a role.
func (r *Repository) PermissionCodes(ctx context.Context, roleID interface{}) ([]string, error) {
	codes := make([]string, 0, 32)
	err := r.db.WithContext(ctx).Raw(`
		SELECT p.code
		FROM permissions p
		JOIN role_permissions rp ON rp.permission_id = p.id
		WHERE rp.role_id = ?
		ORDER BY p.code`, roleID).Scan(&codes).Error
	if err != nil {
		return nil, err
	}
	return codes, nil
}

// OutletsForUser returns the active outlets a user may access. Owners and
// admins receive every outlet in the tenant.
func (r *Repository) OutletsForUser(ctx context.Context, tenantID, userID interface{}, allAccess bool) ([]Outlet, error) {
	outlets := make([]Outlet, 0, 8)
	query := `SELECT id, tenant_id, code, name, address, phone, timezone, is_active, created_at, updated_at
		FROM outlets WHERE tenant_id = ? AND is_active`
	args := []any{tenantID}
	if !allAccess {
		query += ` AND id IN (SELECT outlet_id FROM user_outlets WHERE user_id = ?)`
		args = append(args, userID)
	}
	query += ` ORDER BY name`
	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&outlets).Error; err != nil {
		return nil, err
	}
	return outlets, nil
}

// UserOutletIDs returns assigned outlet IDs for a user.
func (r *Repository) UserOutletIDs(ctx context.Context, userID interface{}) ([]string, error) {
	ids := make([]string, 0, 8)
	err := r.db.WithContext(ctx).Raw(`
		SELECT outlet_id::text FROM user_outlets WHERE user_id = ? ORDER BY created_at`, userID).Scan(&ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// FindRoleBySlug loads a role definition.
func (r *Repository) FindRoleBySlug(ctx context.Context, slug string) (*Role, error) {
	var role Role
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, slug, name, is_system FROM roles WHERE slug = ?`, slug).Scan(&role).Error
	if err != nil {
		return nil, err
	}
	if role.ID.String() == "00000000-0000-0000-0000-000000000000" {
		return nil, ErrNotFound
	}
	return &role, nil
}

// FindRoleByID loads a role definition by primary key.
func (r *Repository) FindRoleByID(ctx context.Context, id interface{}) (*Role, error) {
	var role Role
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, slug, name, is_system FROM roles WHERE id = ?`, id).Scan(&role).Error
	if err != nil {
		return nil, err
	}
	if role.ID.String() == "00000000-0000-0000-0000-000000000000" {
		return nil, ErrNotFound
	}
	return &role, nil
}

// ListRoles returns tenant roles, optionally including super_admin.
func (r *Repository) ListRoles(ctx context.Context, includeSuperAdmin bool) ([]Role, error) {
	roles := make([]Role, 0, 5)
	query := `SELECT id, slug, name, is_system FROM roles`
	if !includeSuperAdmin {
		query += ` WHERE slug <> 'super_admin'`
	}
	query += ` ORDER BY CASE slug
		WHEN 'owner' THEN 1 WHEN 'admin' THEN 2 WHEN 'technician' THEN 3
		WHEN 'cashier' THEN 4 ELSE 5 END`
	if err := r.db.WithContext(ctx).Raw(query).Scan(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// ListPermissions returns permissions, optionally including platform codes.
func (r *Repository) ListPermissions(ctx context.Context, includePlatform bool) ([]Permission, error) {
	permissions := make([]Permission, 0, 48)
	query := `SELECT id, code, module, description FROM permissions`
	if !includePlatform {
		query += ` WHERE module <> 'platform'`
	}
	query += ` ORDER BY module, code`
	if err := r.db.WithContext(ctx).Raw(query).Scan(&permissions).Error; err != nil {
		return nil, err
	}
	return permissions, nil
}

// CountActiveOwners returns the number of active owners in a tenant.
func (r *Repository) CountActiveOwners(ctx context.Context, tenantID interface{}) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Raw(`
		SELECT count(*) FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.tenant_id = ? AND r.slug = 'owner' AND u.is_active = true AND u.deleted_at IS NULL`,
		tenantID).Scan(&count).Error
	return count, err
}

// OutletExistsInTenant validates an outlet belongs to a tenant.
func (r *Repository) OutletExistsInTenant(ctx context.Context, tenantID, outletID interface{}) (bool, error) {
	var exists bool
	err := r.db.WithContext(ctx).Raw(`
		SELECT EXISTS (SELECT 1 FROM outlets WHERE id = ? AND tenant_id = ?)`,
		outletID, tenantID).Scan(&exists).Error
	return exists, err
}

// BumpPermissionVersion invalidates existing access tokens for a user.
func (r *Repository) BumpPermissionVersion(ctx context.Context, tx *gorm.DB, userID interface{}) error {
	h := tx
	if h == nil {
		h = r.db
	}
	return h.WithContext(ctx).Exec(`
		UPDATE users SET permission_version = permission_version + 1, updated_at = now()
		WHERE id = ?`, userID).Error
}

// RevokeUserSessions revokes every active refresh token for a user.
func (r *Repository) RevokeUserSessions(ctx context.Context, tx *gorm.DB, userID interface{}) error {
	h := tx
	if h == nil {
		h = r.db
	}
	return h.WithContext(ctx).Exec(`
		UPDATE refresh_tokens SET revoked_at = now()
		WHERE user_id = ? AND revoked_at IS NULL`, userID).Error
}

// List pagination and filtering.

// RepoListFilter controls the user list query.
type RepoListFilter struct {
	TenantID interface{}
	Query    string
	Role     string
	Status   string
	Page     int
	PerPage  int
}

// ListResult is a paginated user list.
type ListResult struct {
	Items []User
	Total int64
}

// List returns users in a tenant with optional search and role filters.
func (r *Repository) List(ctx context.Context, filter RepoListFilter) (*ListResult, error) {
	where := ` WHERE u.tenant_id = ? AND u.deleted_at IS NULL`
	args := []any{filter.TenantID}
	if filter.Query != "" {
		where += ` AND (u.name ILIKE ? OR u.email ILIKE ? OR u.phone ILIKE ?)`
		like := "%" + filter.Query + "%"
		args = append(args, like, like, like)
	}
	if filter.Role != "" {
		where += ` AND r.slug = ?`
		args = append(args, filter.Role)
	}
	switch filter.Status {
	case "active":
		where += ` AND u.is_active = true`
	case "inactive":
		where += ` AND u.is_active = false`
	}

	var total int64
	if err := r.db.WithContext(ctx).Raw(`
		SELECT count(*) FROM users u JOIN roles r ON r.id = u.role_id`+where, args...).
		Scan(&total).Error; err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}

	listArgs := append(append([]any{}, args...), filter.PerPage, (filter.Page-1)*filter.PerPage)
	items := make([]User, 0, filter.PerPage)
	if err := r.db.WithContext(ctx).Raw(userSelect+where+
		` ORDER BY u.created_at DESC LIMIT ? OFFSET ?`, listArgs...).Scan(&items).Error; err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return &ListResult{Items: items, Total: total}, nil
}

// FindInTenant loads a user only when it belongs to the tenant.
func (r *Repository) FindInTenant(ctx context.Context, tenantID, userID interface{}) (*User, error) {
	var u User
	err := r.db.WithContext(ctx).Raw(userSelect+`
		WHERE u.id = ? AND u.tenant_id = ? AND u.deleted_at IS NULL`, userID, tenantID).Scan(&u).Error
	if err != nil {
		return nil, err
	}
	if u.ID.String() == "00000000-0000-0000-0000-000000000000" {
		return nil, ErrNotFound
	}
	return &u, nil
}

// OutletsByIDs validates and returns outlets in a tenant.
func (r *Repository) OutletsByIDs(ctx context.Context, tenantID interface{}, ids []string) ([]Outlet, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	outlets := make([]Outlet, 0, len(ids))
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, tenant_id, code, name, address, phone, timezone, is_active, created_at, updated_at
		FROM outlets WHERE tenant_id = ? AND id IN ?`, tenantID, ids).Scan(&outlets).Error
	if err != nil {
		return nil, err
	}
	return outlets, nil
}
