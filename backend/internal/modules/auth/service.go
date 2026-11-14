package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"fixphone/backend/internal/database"
	"fixphone/backend/internal/middleware"
	"fixphone/backend/internal/modules/audit"
	"fixphone/backend/internal/modules/user"
	"fixphone/backend/internal/pkg/apperror"
	"fixphone/backend/internal/pkg/cryptoutil"
	"fixphone/backend/internal/pkg/jwtutil"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	refreshCookieName = "fixphone_refresh"
	refreshTokenBytes = 32
)

// ServiceConfig controls authentication policy.
type ServiceConfig struct {
	MaxLoginAttempts int
	LockDuration     time.Duration
	RefreshTTL       time.Duration
}

// Service implements authentication, session, and profile operations.
type Service struct {
	db        *gorm.DB
	users     *user.Repository
	tokens    *jwtutil.Manager
	cfg       ServiceConfig
	dummyHash []byte
}

// NewService builds the auth service.
func NewService(db *gorm.DB, users *user.Repository, tokens *jwtutil.Manager, cfg ServiceConfig) *Service {
	dummy, _ := bcrypt.GenerateFromPassword([]byte("FixPhone#Dummy1"), database.BcryptCost)
	return &Service{db: db, users: users, tokens: tokens, cfg: cfg, dummyHash: dummy}
}

// Login validates credentials, applies account lockout, and starts a session.
func (s *Service) Login(ctx context.Context, input LoginInput, meta Meta) (*Session, error) {
	email, err := database.NormalizeEmail(input.Email)
	if err != nil {
		return nil, s.failedLogin(ctx, nil, input.Email, "format_email", meta)
	}

	u, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			// Equalize timing with a real bcrypt comparison.
			_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(input.Password))
			return nil, s.failedLogin(ctx, nil, email, "user_tidak_ditemukan", meta)
		}
		return nil, apperror.Internal(err)
	}

	now := time.Now().UTC()
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(input.Password)) != nil {
		if u.LockedUntil != nil && u.LockedUntil.After(now) {
			return nil, s.failedLogin(ctx, u, email, "akun_terkunci", meta)
		}
		return nil, s.recordFailedAttempt(ctx, u, meta, now)
	}

	if u.LockedUntil != nil && u.LockedUntil.After(now) {
		audit.BestEffort(ctx, s.db, audit.Entry{
			TenantID: u.TenantID, UserID: &u.ID, ActorName: u.Name, Action: "auth.login_blocked",
			Module: "auth", EntityType: "user", EntityID: &u.ID, IP: meta.IP,
			UserAgent: meta.UserAgent, RequestID: meta.RequestID,
		})
		return nil, apperror.New(429, "ACCOUNT_LOCKED", "Akun terkunci sementara karena percobaan login gagal. Coba lagi nanti.")
	}

	if !u.IsActive {
		return nil, s.failedLogin(ctx, u, email, "akun_nonaktif", meta)
	}
	if err := s.ensureTenantActive(ctx, u.TenantID); err != nil {
		return nil, err
	}

	if err := s.db.WithContext(ctx).Exec(`
		UPDATE users SET failed_login_count = 0, locked_until = NULL, last_login_at = now(), updated_at = now()
		WHERE id = ?`, u.ID).Error; err != nil {
		return nil, apperror.Internal(err)
	}
	u.FailedLoginCount = 0
	u.LockedUntil = nil
	u.LastLoginAt = &now

	session, err := s.startSession(ctx, u, meta)
	if err != nil {
		return nil, err
	}

	audit.BestEffort(ctx, s.db, audit.Entry{
		TenantID: u.TenantID, UserID: &u.ID, ActorName: u.Name, Action: "auth.login",
		Module: "auth", EntityType: "user", EntityID: &u.ID, IP: meta.IP,
		UserAgent: meta.UserAgent, RequestID: meta.RequestID,
	})
	return session, nil
}

func (s *Service) recordFailedAttempt(ctx context.Context, u *user.User, meta Meta, now time.Time) error {
	failures := u.FailedLoginCount + 1
	if u.LockedUntil != nil && !u.LockedUntil.After(now) {
		failures = 1
	}
	var lockedUntil *time.Time
	if failures >= s.cfg.MaxLoginAttempts {
		until := now.Add(s.cfg.LockDuration)
		lockedUntil = &until
	}
	if err := s.db.WithContext(ctx).Exec(`
		UPDATE users SET failed_login_count = ?, locked_until = ?, updated_at = now()
		WHERE id = ?`, failures, lockedUntil, u.ID).Error; err != nil {
		return apperror.Internal(err)
	}

	audit.BestEffort(ctx, s.db, audit.Entry{
		TenantID: u.TenantID, UserID: &u.ID, ActorName: u.Name, Action: "auth.login_failed",
		Module: "auth", EntityType: "user", EntityID: &u.ID,
		After: map[string]any{"failed_attempts": failures}, IP: meta.IP,
		UserAgent: meta.UserAgent, RequestID: meta.RequestID,
	})
	if lockedUntil != nil {
		audit.BestEffort(ctx, s.db, audit.Entry{
			TenantID: u.TenantID, UserID: &u.ID, ActorName: u.Name, Action: "auth.lockout",
			Module: "auth", EntityType: "user", EntityID: &u.ID,
			After: map[string]any{"locked_until": lockedUntil.Format(time.RFC3339), "failed_attempts": failures},
			IP:    meta.IP, UserAgent: meta.UserAgent, RequestID: meta.RequestID,
		})
	}
	return apperror.InvalidCredentials()
}

func (s *Service) failedLogin(ctx context.Context, u *user.User, email, reason string, meta Meta) error {
	entry := audit.Entry{
		Action: "auth.login_failed", Module: "auth", EntityType: "auth",
		After: map[string]any{"identifier": email, "reason": reason},
		IP:    meta.IP, UserAgent: meta.UserAgent, RequestID: meta.RequestID,
	}
	if u != nil {
		entry.TenantID = u.TenantID
		entry.UserID = &u.ID
		entry.ActorName = u.Name
		entry.EntityID = &u.ID
	}
	audit.BestEffort(ctx, s.db, entry)
	return apperror.InvalidCredentials()
}

func (s *Service) startSession(ctx context.Context, u *user.User, meta Meta) (*Session, error) {
	familyID, err := uuid.NewV7()
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return s.issueSession(ctx, u, familyID, meta)
}

func (s *Service) issueSession(ctx context.Context, u *user.User, familyID uuid.UUID, meta Meta) (*Session, error) {
	rawToken, err := cryptoutil.RandomToken(refreshTokenBytes)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	tokenID, err := uuid.NewV7()
	if err != nil {
		return nil, apperror.Internal(err)
	}
	refreshExpires := time.Now().UTC().Add(s.cfg.RefreshTTL)

	if err := s.db.WithContext(ctx).Exec(`
		INSERT INTO refresh_tokens (id, user_id, token_hash, family_id, expires_at, ip, user_agent)
		VALUES (?, ?, ?, ?, ?, NULLIF(?, '')::inet, NULLIF(?, ''))`,
		tokenID, u.ID, cryptoutil.HashToken(rawToken), familyID, refreshExpires,
		meta.IP, truncate(meta.UserAgent, 255)).Error; err != nil {
		return nil, apperror.Internal(err)
	}

	accessToken, _, err := s.tokens.Issue(u.ID, u.TenantID, u.RoleSlug, familyID.String(), u.PermissionVersion)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	profile, err := s.buildProfile(ctx, u)
	if err != nil {
		return nil, err
	}
	return &Session{
		AccessToken:         accessToken,
		ExpiresIn:           int(s.tokens.TTL().Seconds()),
		RefreshToken:        rawToken,
		RefreshTokenExpires: refreshExpires,
		User:                profile,
	}, nil
}

// Refresh rotates a refresh token and detects reuse.
func (s *Service) Refresh(ctx context.Context, rawToken string, meta Meta) (*Session, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return nil, apperror.Unauthorized("INVALID_REFRESH_TOKEN", "Sesi tidak ditemukan. Silakan login kembali.")
	}
	hash := cryptoutil.HashToken(rawToken)

	var record user.RefreshToken
	var reuse bool
	var revokedFamily uuid.UUID
	var newSession *Session

	err := database.WithContext(ctx, s.db).Transaction(func(tx *gorm.DB) error {
		err := tx.Raw(`
			SELECT id, user_id, token_hash, family_id, expires_at, revoked_at, replaced_by_id, ip, user_agent, created_at
			FROM refresh_tokens WHERE token_hash = ? FOR UPDATE`, hash).Scan(&record).Error
		if err != nil {
			return err
		}
		if record.ID == uuid.Nil {
			return apperror.Unauthorized("INVALID_REFRESH_TOKEN", "Sesi tidak valid. Silakan login kembali.")
		}
		if record.RevokedAt != nil {
			reuse = true
			revokedFamily = record.FamilyID
			if err := tx.Exec(`UPDATE refresh_tokens SET revoked_at = now()
				WHERE family_id = ? AND revoked_at IS NULL`, record.FamilyID).Error; err != nil {
				return err
			}
			// Commit the family revocation. The handler turns the reuse flag
			// into an authentication error after the transaction.
			return nil
		}
		if time.Now().UTC().After(record.ExpiresAt) {
			if err := tx.Exec(`UPDATE refresh_tokens SET revoked_at = now() WHERE id = ?`, record.ID).Error; err != nil {
				return err
			}
			return apperror.Unauthorized("INVALID_REFRESH_TOKEN", "Sesi telah berakhir. Silakan login kembali.")
		}

		u, err := s.users.FindByID(ctx, record.UserID)
		if err != nil || !u.IsActive {
			return apperror.Unauthorized("INVALID_REFRESH_TOKEN", "Sesi tidak valid. Silakan login kembali.")
		}
		if err := s.ensureTenantActive(ctx, u.TenantID); err != nil {
			return err
		}

		rawNew, err := cryptoutil.RandomToken(refreshTokenBytes)
		if err != nil {
			return err
		}
		newID, err := uuid.NewV7()
		if err != nil {
			return err
		}
		refreshExpires := time.Now().UTC().Add(s.cfg.RefreshTTL)
		if err := tx.Exec(`
			INSERT INTO refresh_tokens (id, user_id, token_hash, family_id, expires_at, ip, user_agent)
			VALUES (?, ?, ?, ?, ?, NULLIF(?, '')::inet, NULLIF(?, ''))`,
			newID, u.ID, cryptoutil.HashToken(rawNew), record.FamilyID, refreshExpires,
			meta.IP, truncate(meta.UserAgent, 255)).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			UPDATE refresh_tokens SET revoked_at = now(), replaced_by_id = ? WHERE id = ?`,
			newID, record.ID).Error; err != nil {
			return err
		}

		accessToken, _, err := s.tokens.Issue(u.ID, u.TenantID, u.RoleSlug, record.FamilyID.String(), u.PermissionVersion)
		if err != nil {
			return err
		}
		profile, err := s.buildProfile(ctx, u)
		if err != nil {
			return err
		}
		newSession = &Session{
			AccessToken:         accessToken,
			ExpiresIn:           int(s.tokens.TTL().Seconds()),
			RefreshToken:        rawNew,
			RefreshTokenExpires: refreshExpires,
			User:                profile,
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	if reuse {
		entry := audit.Entry{
			Action: "auth.token_reuse", Module: "auth", EntityType: "refresh_token",
			After: map[string]any{"family_id": revokedFamily.String()},
			IP:    meta.IP, UserAgent: meta.UserAgent, RequestID: meta.RequestID,
		}
		if u, findErr := s.users.FindByID(ctx, record.UserID); findErr == nil {
			entry.TenantID = u.TenantID
			entry.UserID = &u.ID
			entry.ActorName = u.Name
		}
		audit.BestEffort(ctx, s.db, entry)
		return nil, apperror.Unauthorized("TOKEN_REUSED", "Sesi tidak valid karena token digunakan ulang. Silakan login kembali.")
	}
	return newSession, nil
}

// Logout revokes the token family for the current refresh token or session.
func (s *Service) Logout(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, rawToken string, meta Meta) error {
	familyID := sessionID
	rawToken = strings.TrimSpace(rawToken)
	if rawToken != "" {
		var found string
		if err := s.db.WithContext(ctx).Raw(`
			SELECT family_id::text FROM refresh_tokens WHERE token_hash = ?`,
			cryptoutil.HashToken(rawToken)).Scan(&found).Error; err == nil && found != "" {
			if parsed, parseErr := uuid.Parse(found); parseErr == nil {
				familyID = parsed
			}
		}
	}
	if familyID != uuid.Nil {
		if err := s.db.WithContext(ctx).Exec(`
			UPDATE refresh_tokens SET revoked_at = now()
			WHERE family_id = ? AND revoked_at IS NULL`, familyID).Error; err != nil {
			return apperror.Internal(err)
		}
	}

	u, _ := s.users.FindByID(ctx, userID)
	entry := audit.Entry{
		UserID: &userID, Action: "auth.logout", Module: "auth", EntityType: "user",
		EntityID: &userID, IP: meta.IP, UserAgent: meta.UserAgent, RequestID: meta.RequestID,
	}
	if u != nil {
		entry.TenantID = u.TenantID
		entry.ActorName = u.Name
	}
	audit.BestEffort(ctx, s.db, entry)
	return nil
}

// Me returns the fresh profile for an authenticated user.
func (s *Service) Me(ctx context.Context, userID uuid.UUID) (*Profile, error) {
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return nil, apperror.Unauthorized("UNAUTHORIZED", "Pengguna tidak ditemukan.")
		}
		return nil, apperror.Internal(err)
	}
	return s.buildProfile(ctx, u)
}

// UpdateProfile edits name, phone, and avatar path.
func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, input ProfileInput, meta Meta) (*Profile, error) {
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	name := strings.TrimSpace(input.Name)
	if name == "" || len(name) > 120 {
		return nil, apperror.Validation(map[string]string{"name": "Nama wajib diisi, maksimal 120 karakter."})
	}
	phone := trimOptional(input.Phone)
	if phone != nil && len(*phone) > 30 {
		return nil, apperror.Validation(map[string]string{"phone": "Nomor telepon maksimal 30 karakter."})
	}
	avatar := trimOptional(input.AvatarPath)
	if avatar != nil {
		value := strings.ReplaceAll(*avatar, "\\", "/")
		if len(value) > 255 || strings.Contains(value, "..") || (!strings.HasPrefix(value, "uploads/") && !strings.HasPrefix(value, "/uploads/")) {
			return nil, apperror.Validation(map[string]string{"avatar_path": "Path avatar tidak valid."})
		}
		avatar = &value
	}

	before := map[string]any{"name": u.Name, "phone": u.Phone, "avatar_path": u.AvatarPath}
	if err := s.db.WithContext(ctx).Exec(`
		UPDATE users SET name = ?, phone = ?, avatar_path = ?, updated_at = now() WHERE id = ?`,
		name, phone, avatar, userID).Error; err != nil {
		return nil, apperror.Internal(err)
	}

	u.Name, u.Phone, u.AvatarPath = name, phone, avatar
	audit.BestEffort(ctx, s.db, audit.Entry{
		TenantID: u.TenantID, UserID: &u.ID, ActorName: u.Name, Action: "auth.profile_updated",
		Module: "auth", EntityType: "user", EntityID: &u.ID,
		Before: before, After: map[string]any{"name": name, "phone": phone, "avatar_path": avatar},
		IP: meta.IP, UserAgent: meta.UserAgent, RequestID: meta.RequestID,
	})
	return s.buildProfile(ctx, u)
}

// ChangePassword verifies the current password, stores a new hash, and revokes
// every session (including the current one).
func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, input ChangePasswordInput, meta Meta) error {
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return apperror.Internal(err)
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(input.CurrentPassword)) != nil {
		return apperror.New(422, "INVALID_CURRENT_PASSWORD", "Password lama tidak sesuai.")
	}
	if input.NewPassword == input.CurrentPassword {
		return apperror.Validation(map[string]string{"new_password": "Password baru harus berbeda dari password lama."})
	}
	if err := database.ValidatePassword(input.NewPassword); err != nil {
		return apperror.Validation(map[string]string{"new_password": err.Error()})
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), database.BcryptCost)
	if err != nil {
		return apperror.Internal(err)
	}
	err = database.WithContext(ctx, s.db).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			UPDATE users
			SET password_hash = ?, must_change_password = false,
			    permission_version = permission_version + 1, updated_at = now()
			WHERE id = ?`, string(hash), userID).Error; err != nil {
			return err
		}
		return tx.Exec(`UPDATE refresh_tokens SET revoked_at = now()
			WHERE user_id = ? AND revoked_at IS NULL`, userID).Error
	})
	if err != nil {
		return apperror.Internal(err)
	}

	audit.BestEffort(ctx, s.db, audit.Entry{
		TenantID: u.TenantID, UserID: &u.ID, ActorName: u.Name, Action: "auth.password_changed",
		Module: "auth", EntityType: "user", EntityID: &u.ID,
		IP: meta.IP, UserAgent: meta.UserAgent, RequestID: meta.RequestID,
	})
	return nil
}

// LoadAuthContext implements middleware.IdentityLoader.
func (s *Service) LoadAuthContext(ctx context.Context, userID uuid.UUID, permissionVersion int) (*middleware.AuthContext, error) {
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return nil, apperror.Unauthorized("UNAUTHORIZED", "Sesi tidak valid. Silakan login kembali.")
		}
		return nil, apperror.Internal(err)
	}
	if !u.IsActive {
		return nil, apperror.Unauthorized("ACCOUNT_INACTIVE", "Akun dinonaktifkan. Hubungi administrator.")
	}
	if u.PermissionVersion != permissionVersion {
		return nil, apperror.Unauthorized("TOKEN_INVALID", "Sesi tidak lagi valid. Silakan login kembali.")
	}
	codes, err := s.users.PermissionCodes(ctx, u.RoleID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	permissions := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		permissions[code] = struct{}{}
	}
	return &middleware.AuthContext{
		UserID:             u.ID,
		TenantID:           u.TenantID,
		Name:               u.Name,
		Email:              u.Email,
		Role:               u.RoleSlug,
		Permissions:        permissions,
		MustChangePassword: u.MustChangePassword,
		PermissionVersion:  u.PermissionVersion,
	}, nil
}

func (s *Service) buildProfile(ctx context.Context, u *user.User) (*Profile, error) {
	codes, err := s.users.PermissionCodes(ctx, u.RoleID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	profile := &Profile{
		ID:                 u.ID.String(),
		Name:               u.Name,
		Email:              u.Email,
		Phone:              u.Phone,
		AvatarPath:         u.AvatarPath,
		Role:               u.RoleSlug,
		Permissions:        codes,
		Outlets:            []Outlet{},
		MustChangePassword: u.MustChangePassword,
		LastLoginAt:        u.LastLoginAt,
	}
	if u.TenantID == nil {
		return profile, nil
	}
	tenantID := u.TenantID.String()
	profile.TenantID = &tenantID

	allAccess := u.RoleSlug == "owner" || u.RoleSlug == "admin"
	outlets, err := s.users.OutletsForUser(ctx, u.TenantID, u.ID, allAccess)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	profile.Outlets = userOutlets(outlets)
	if len(profile.Outlets) > 0 {
		active := profile.Outlets[0].ID
		profile.ActiveOutletID = &active
	}
	return profile, nil
}

func (s *Service) ensureTenantActive(ctx context.Context, tenantID *uuid.UUID) error {
	if tenantID == nil {
		return nil
	}
	var status string
	if err := s.db.WithContext(ctx).Raw(`SELECT status FROM tenants WHERE id = ?`, *tenantID).Scan(&status).Error; err != nil {
		return apperror.Internal(err)
	}
	switch status {
	case "":
		return apperror.Unauthorized("TENANT_NOT_FOUND", "Tenant tidak ditemukan.")
	case "suspended":
		return apperror.Forbidden("TENANT_SUSPENDED", "Akun toko sedang dinonaktifkan.")
	}
	return nil
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

func truncate(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return value[:max]
}
