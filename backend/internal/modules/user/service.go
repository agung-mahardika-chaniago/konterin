package user

import (
	"context"
	"errors"
	"strings"

	"fixphone/backend/internal/database"
	"fixphone/backend/internal/modules/audit"
	"fixphone/backend/internal/pkg/apperror"
	"fixphone/backend/internal/pkg/cryptoutil"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Service implements tenant-scoped user management.
type Service struct {
	db   *gorm.DB
	repo *Repository
}

// NewService builds the user service.
func NewService(db *gorm.DB, repo *Repository) *Service {
	return &Service{db: db, repo: repo}
}

// List returns a paginated user list for the actor's tenant.
func (s *Service) List(ctx context.Context, actor Actor, filter ListFilter) (*ListResponse, error) {
	tenantID, err := tenantUUID(actor)
	if err != nil {
		return nil, err
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 || filter.PerPage > 100 {
		filter.PerPage = 20
	}
	result, err := s.repo.List(ctx, RepoListFilter{
		TenantID: tenantID, Query: strings.TrimSpace(filter.Query),
		Role: filter.Role, Status: filter.Status, Page: filter.Page, PerPage: filter.PerPage,
	})
	if err != nil {
		return nil, apperror.Internal(err)
	}
	outlets, err := s.outletMap(ctx, result.Items)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	items := make([]UserDTO, 0, len(result.Items))
	for i := range result.Items {
		u := result.Items[i]
		items = append(items, toDTO(u, outlets[u.ID]))
	}
	return &ListResponse{Items: items, Total: result.Total}, nil
}

// Get returns one user in the actor's tenant.
func (s *Service) Get(ctx context.Context, actor Actor, id uuid.UUID) (*UserDTO, error) {
	tenantID, err := tenantUUID(actor)
	if err != nil {
		return nil, err
	}
	u, err := s.repo.FindInTenant(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, apperror.NotFound("Pengguna tidak ditemukan.")
		}
		return nil, apperror.Internal(err)
	}
	outlets, err := s.outletsForUser(ctx, u)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	dto := toDTO(*u, outlets)
	return &dto, nil
}

// Create adds a tenant user.
func (s *Service) Create(ctx context.Context, actor Actor, input CreateInput) (*UserDTO, error) {
	tenantID, err := tenantUUID(actor)
	if err != nil {
		return nil, err
	}
	role, err := s.validateRole(ctx, input.Role)
	if err != nil {
		return nil, err
	}
	if !canManageRole(actor.Role, role.Slug) {
		return nil, apperror.Forbidden("FORBIDDEN", "Anda tidak berwenang membuat pengguna dengan role tersebut.")
	}

	email, err := database.NormalizeEmail(input.Email)
	if err != nil {
		return nil, apperror.Validation(map[string]string{"email": err.Error()})
	}
	name := strings.TrimSpace(input.Name)
	if name == "" || len(name) > 120 {
		return nil, apperror.Validation(map[string]string{"name": "Nama wajib diisi, maksimal 120 karakter."})
	}
	phone := trimOptional(input.Phone)
	if phone != nil && len(*phone) > 30 {
		return nil, apperror.Validation(map[string]string{"phone": "Nomor telepon maksimal 30 karakter."})
	}
	if err := database.ValidatePassword(input.Password); err != nil {
		return nil, apperror.Validation(map[string]string{"password": err.Error()})
	}

	exists, err := s.emailExists(ctx, email)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if exists {
		return nil, apperror.Conflict("CONFLICT", "Email sudah digunakan.")
	}

	outletIDs, err := s.validateOutlets(ctx, tenantID, input.OutletIDs, role.Slug)
	if err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), database.BcryptCost)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	userID, err := uuid.NewV7()
	if err != nil {
		return nil, apperror.Internal(err)
	}
	isActive := true
	if input.IsActive == false {
		isActive = false
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			INSERT INTO users (id, tenant_id, role_id, name, email, phone, password_hash,
			                   avatar_path, is_active, must_change_password)
			VALUES (?, ?, ?, ?, ?, ?, ?, NULL, ?, true)`,
			userID, tenantID, role.ID, name, email, phone, string(hash), isActive).Error; err != nil {
			return err
		}
		for _, outletID := range outletIDs {
			if err := tx.Exec(`INSERT INTO user_outlets (user_id, outlet_id) VALUES (?, ?)`,
				userID, outletID).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, apperror.Conflict("CONFLICT", "Email sudah digunakan.")
		}
		return nil, apperror.Internal(err)
	}

	audit.BestEffort(ctx, s.db, audit.Entry{
		TenantID: &tenantID, UserID: actorUUID(actor), ActorName: actor.Name,
		Action: "user.create", Module: "user", EntityType: "user", EntityID: &userID,
		After: map[string]any{"name": name, "email": email, "role": role.Slug, "is_active": isActive},
	})
	return s.Get(ctx, actor, userID)
}

// Update edits a tenant user.
func (s *Service) Update(ctx context.Context, actor Actor, id uuid.UUID, input UpdateInput) (*UserDTO, error) {
	tenantID, err := tenantUUID(actor)
	if err != nil {
		return nil, err
	}
	current, err := s.repo.FindInTenant(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, apperror.NotFound("Pengguna tidak ditemukan.")
		}
		return nil, apperror.Internal(err)
	}
	if !canManageRole(actor.Role, current.RoleSlug) {
		return nil, apperror.Forbidden("FORBIDDEN", "Anda tidak berwenang mengubah pengguna dengan role tersebut.")
	}

	name := current.Name
	if input.Name != nil {
		name = strings.TrimSpace(*input.Name)
		if name == "" || len(name) > 120 {
			return nil, apperror.Validation(map[string]string{"name": "Nama wajib diisi, maksimal 120 karakter."})
		}
	}
	phone := current.Phone
	if input.Phone != nil {
		phone = trimOptional(input.Phone)
		if phone != nil && len(*phone) > 30 {
			return nil, apperror.Validation(map[string]string{"phone": "Nomor telepon maksimal 30 karakter."})
		}
	}

	role := &Role{ID: current.RoleID, Slug: current.RoleSlug, Name: current.RoleName}
	roleChanged := false
	if input.Role != nil && strings.TrimSpace(*input.Role) != current.RoleSlug {
		nextRole, err := s.validateRole(ctx, *input.Role)
		if err != nil {
			return nil, err
		}
		if !canManageRole(actor.Role, nextRole.Slug) {
			return nil, apperror.Forbidden("FORBIDDEN", "Anda tidak berwenang memberikan role tersebut.")
		}
		role = nextRole
		roleChanged = true
	}

	isActive := current.IsActive
	deactivated := false
	if input.IsActive != nil && *input.IsActive != current.IsActive {
		isActive = *input.IsActive
		deactivated = !isActive
	}
	if deactivated {
		if current.ID.String() == actor.UserID {
			return nil, apperror.Conflict("CANNOT_DEACTIVATE_SELF", "Anda tidak dapat menonaktifkan akun sendiri.")
		}
		if current.RoleSlug == "owner" {
			owners, err := s.repo.CountActiveOwners(ctx, tenantID)
			if err != nil {
				return nil, apperror.Internal(err)
			}
			if owners <= 1 {
				return nil, apperror.Conflict("LAST_OWNER", "Owner terakhir tidak dapat dinonaktifkan.")
			}
		}
	}

	before := map[string]any{"name": current.Name, "phone": current.Phone, "role": current.RoleSlug, "is_active": current.IsActive}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			UPDATE users SET name = ?, phone = ?, role_id = ?, is_active = ?, updated_at = now()
			WHERE id = ?`, name, phone, role.ID, isActive, current.ID).Error; err != nil {
			return err
		}
		if roleChanged || deactivated {
			if err := tx.Exec(`
				UPDATE users SET permission_version = permission_version + 1 WHERE id = ?`, current.ID).Error; err != nil {
				return err
			}
			if err := tx.Exec(`
				UPDATE refresh_tokens SET revoked_at = now()
				WHERE user_id = ? AND revoked_at IS NULL`, current.ID).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, apperror.Internal(err)
	}

	audit.BestEffort(ctx, s.db, audit.Entry{
		TenantID: &tenantID, UserID: actorUUID(actor), ActorName: actor.Name,
		Action: "user.update", Module: "user", EntityType: "user", EntityID: &current.ID,
		Before: before, After: map[string]any{"name": name, "phone": phone, "role": role.Slug, "is_active": isActive},
	})
	return s.Get(ctx, actor, current.ID)
}

// Delete soft-deletes a tenant user and revokes their sessions.
func (s *Service) Delete(ctx context.Context, actor Actor, id uuid.UUID) error {
	tenantID, err := tenantUUID(actor)
	if err != nil {
		return err
	}
	current, err := s.repo.FindInTenant(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return apperror.NotFound("Pengguna tidak ditemukan.")
		}
		return apperror.Internal(err)
	}
	if current.ID.String() == actor.UserID {
		return apperror.Conflict("CANNOT_DELETE_SELF", "Anda tidak dapat menghapus akun sendiri.")
	}
	if !canManageRole(actor.Role, current.RoleSlug) {
		return apperror.Forbidden("FORBIDDEN", "Anda tidak berwenang menghapus pengguna dengan role tersebut.")
	}
	if current.RoleSlug == "owner" {
		owners, err := s.repo.CountActiveOwners(ctx, tenantID)
		if err != nil {
			return apperror.Internal(err)
		}
		if owners <= 1 {
			return apperror.Conflict("LAST_OWNER", "Owner terakhir tidak dapat dihapus.")
		}
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			UPDATE users
			SET deleted_at = now(), is_active = false,
			    permission_version = permission_version + 1, updated_at = now()
			WHERE id = ?`, current.ID).Error; err != nil {
			return err
		}
		return tx.Exec(`UPDATE refresh_tokens SET revoked_at = now()
			WHERE user_id = ? AND revoked_at IS NULL`, current.ID).Error
	})
	if err != nil {
		return apperror.Internal(err)
	}
	audit.BestEffort(ctx, s.db, audit.Entry{
		TenantID: &tenantID, UserID: actorUUID(actor), ActorName: actor.Name,
		Action: "user.delete", Module: "user", EntityType: "user", EntityID: &current.ID,
		Before: map[string]any{"name": current.Name, "email": current.Email},
	})
	return nil
}

// ResetPassword sets a temporary password and returns it exactly once.
func (s *Service) ResetPassword(ctx context.Context, actor Actor, id uuid.UUID) (*ResetPasswordResponse, error) {
	tenantID, err := tenantUUID(actor)
	if err != nil {
		return nil, err
	}
	current, err := s.repo.FindInTenant(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, apperror.NotFound("Pengguna tidak ditemukan.")
		}
		return nil, apperror.Internal(err)
	}
	if current.ID.String() == actor.UserID {
		return nil, apperror.Conflict("USE_CHANGE_PASSWORD", "Gunakan menu ubah password untuk akun sendiri.")
	}
	if !canManageRole(actor.Role, current.RoleSlug) {
		return nil, apperror.Forbidden("FORBIDDEN", "Anda tidak berwenang mereset password pengguna ini.")
	}

	tempPassword, err := cryptoutil.RandomPassword(12)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(tempPassword), database.BcryptCost)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			UPDATE users
			SET password_hash = ?, must_change_password = true,
			    failed_login_count = 0, locked_until = NULL,
			    permission_version = permission_version + 1, updated_at = now()
			WHERE id = ?`, string(hash), current.ID).Error; err != nil {
			return err
		}
		return tx.Exec(`UPDATE refresh_tokens SET revoked_at = now()
			WHERE user_id = ? AND revoked_at IS NULL`, current.ID).Error
	})
	if err != nil {
		return nil, apperror.Internal(err)
	}
	audit.BestEffort(ctx, s.db, audit.Entry{
		TenantID: &tenantID, UserID: actorUUID(actor), ActorName: actor.Name,
		Action: "user.password_reset", Module: "user", EntityType: "user", EntityID: &current.ID,
	})
	return &ResetPasswordResponse{TemporaryPassword: tempPassword, MustChangePassword: true}, nil
}

// ReplaceOutlets replaces the outlet assignments of a tenant user.
func (s *Service) ReplaceOutlets(ctx context.Context, actor Actor, id uuid.UUID, outletIDs []string) (*UserDTO, error) {
	tenantID, err := tenantUUID(actor)
	if err != nil {
		return nil, err
	}
	current, err := s.repo.FindInTenant(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, apperror.NotFound("Pengguna tidak ditemukan.")
		}
		return nil, apperror.Internal(err)
	}
	if !canManageRole(actor.Role, current.RoleSlug) {
		return nil, apperror.Forbidden("FORBIDDEN", "Anda tidak berwenang mengubah outlet pengguna ini.")
	}
	valid, err := s.validateOutlets(ctx, tenantID, outletIDs, current.RoleSlug)
	if err != nil {
		return nil, err
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`DELETE FROM user_outlets WHERE user_id = ?`, current.ID).Error; err != nil {
			return err
		}
		for _, outletID := range valid {
			if err := tx.Exec(`INSERT INTO user_outlets (user_id, outlet_id) VALUES (?, ?)`,
				current.ID, outletID).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, apperror.Internal(err)
	}
	audit.BestEffort(ctx, s.db, audit.Entry{
		TenantID: &tenantID, UserID: actorUUID(actor), ActorName: actor.Name,
		Action: "user.outlets_update", Module: "user", EntityType: "user", EntityID: &current.ID,
		After: map[string]any{"outlet_ids": outletIDs},
	})
	return s.Get(ctx, actor, current.ID)
}

// Roles returns tenant roles with their permission codes.
func (s *Service) Roles(ctx context.Context, actor Actor) ([]RoleDTO, error) {
	if _, err := tenantUUID(actor); err != nil {
		return nil, err
	}
	roles, err := s.repo.ListRoles(ctx, false)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	out := make([]RoleDTO, 0, len(roles))
	for _, role := range roles {
		codes, err := s.repo.PermissionCodes(ctx, role.ID)
		if err != nil {
			return nil, apperror.Internal(err)
		}
		out = append(out, RoleDTO{ID: role.ID.String(), Slug: role.Slug, Name: role.Name, Permissions: codes})
	}
	return out, nil
}

// Permissions returns tenant permission definitions.
func (s *Service) Permissions(ctx context.Context, actor Actor) ([]PermissionDTO, error) {
	if _, err := tenantUUID(actor); err != nil {
		return nil, err
	}
	permissions, err := s.repo.ListPermissions(ctx, false)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	out := make([]PermissionDTO, 0, len(permissions))
	for _, permission := range permissions {
		out = append(out, PermissionDTO{Code: permission.Code, Module: permission.Module, Description: permission.Description})
	}
	return out, nil
}

func (s *Service) validateRole(ctx context.Context, slug string) (*Role, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, apperror.Validation(map[string]string{"role": "Role wajib dipilih."})
	}
	if slug == "super_admin" {
		return nil, apperror.Validation(map[string]string{"role": "Role super admin tidak dapat diberikan melalui tenant."})
	}
	role, err := s.repo.FindRoleBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, apperror.Validation(map[string]string{"role": "Role tidak dikenal."})
		}
		return nil, apperror.Internal(err)
	}
	return role, nil
}

func (s *Service) validateOutlets(ctx context.Context, tenantID uuid.UUID, raw []string, role string) ([]uuid.UUID, error) {
	seen := make(map[uuid.UUID]struct{}, len(raw))
	out := make([]uuid.UUID, 0, len(raw))
	for _, value := range raw {
		id, err := uuid.Parse(strings.TrimSpace(value))
		if err != nil {
			return nil, apperror.Validation(map[string]string{"outlet_ids": "ID outlet tidak valid."})
		}
		if _, exists := seen[id]; exists {
			continue
		}
		found, err := s.repo.OutletExistsInTenant(ctx, tenantID, id)
		if err != nil {
			return nil, apperror.Internal(err)
		}
		if !found {
			return nil, apperror.Validation(map[string]string{"outlet_ids": "Outlet tidak ditemukan pada tenant ini."})
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) == 0 && (role == "technician" || role == "cashier") {
		return nil, apperror.Validation(map[string]string{"outlet_ids": "Pilih minimal satu outlet untuk role ini."})
	}
	return out, nil
}

func (s *Service) emailExists(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := s.db.WithContext(ctx).Raw(`
		SELECT EXISTS (SELECT 1 FROM users WHERE email = ? AND deleted_at IS NULL)`, email).Scan(&exists).Error
	return exists, err
}

func (s *Service) outletsForUser(ctx context.Context, u *User) ([]Outlet, error) {
	allAccess := u.RoleSlug == "owner" || u.RoleSlug == "admin"
	return s.repo.OutletsForUser(ctx, *u.TenantID, u.ID, allAccess)
}

func (s *Service) outletMap(ctx context.Context, items []User) (map[uuid.UUID][]Outlet, error) {
	result := make(map[uuid.UUID][]Outlet, len(items))
	if len(items) == 0 {
		return result, nil
	}

	type assigned struct {
		UserID string    `gorm:"column:user_id"`
		ID     uuid.UUID `gorm:"column:id"`
		Code   string    `gorm:"column:code"`
		Name   string    `gorm:"column:name"`
	}
	userIDs := make([]string, 0, len(items))
	for _, u := range items {
		userIDs = append(userIDs, u.ID.String())
	}
	rows := make([]assigned, 0)
	if err := s.db.WithContext(ctx).Raw(`
		SELECT uo.user_id::text AS user_id, o.id, o.code, o.name
		FROM user_outlets uo
		JOIN outlets o ON o.id = uo.outlet_id
		WHERE uo.user_id IN ?
		ORDER BY o.name`, userIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	byUser := make(map[string][]Outlet)
	for _, row := range rows {
		byUser[row.UserID] = append(byUser[row.UserID], Outlet{ID: row.ID, Code: row.Code, Name: row.Name})
	}

	var allOutlets []Outlet
	allLoaded := false
	for _, u := range items {
		if u.RoleSlug == "owner" || u.RoleSlug == "admin" {
			if !allLoaded {
				outlets, err := s.repo.OutletsForUser(ctx, *u.TenantID, u.ID, true)
				if err != nil {
					return nil, err
				}
				allOutlets = outlets
				allLoaded = true
			}
			result[u.ID] = allOutlets
			continue
		}
		result[u.ID] = byUser[u.ID.String()]
	}
	return result, nil
}

func toDTO(u User, outlets []Outlet) UserDTO {
	dto := UserDTO{
		ID:                 u.ID.String(),
		Name:               u.Name,
		Email:              u.Email,
		Phone:              u.Phone,
		Role:               u.RoleSlug,
		RoleName:           u.RoleName,
		IsActive:           u.IsActive,
		MustChangePassword: u.MustChangePassword,
		LastLoginAt:        u.LastLoginAt,
		Outlets:            make([]OutletDTO, 0, len(outlets)),
		CreatedAt:          u.CreatedAt,
		UpdatedAt:          u.UpdatedAt,
	}
	for _, outlet := range outlets {
		dto.Outlets = append(dto.Outlets, OutletDTO{ID: outlet.ID.String(), Code: outlet.Code, Name: outlet.Name})
	}
	return dto
}

func canManageRole(actorRole, targetRole string) bool {
	if targetRole == "super_admin" {
		return false
	}
	switch actorRole {
	case "owner":
		return true
	case "admin":
		return targetRole != "owner"
	default:
		return false
	}
}

func tenantUUID(actor Actor) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(actor.TenantID))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, apperror.Forbidden("FORBIDDEN", "Endpoint ini hanya tersedia untuk pengguna tenant.")
	}
	return id, nil
}

func actorUUID(actor Actor) *uuid.UUID {
	id, err := uuid.Parse(actor.UserID)
	if err != nil {
		return nil
	}
	return &id
}

func trimOptional(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return strings.Contains(err.Error(), "23505")
}
