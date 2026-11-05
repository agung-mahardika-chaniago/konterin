// Package jwtutil issues and verifies the HS256 access tokens used by the API.
//
// The implementation is intentionally small and dependency-free. Verification
// pins the algorithm to HS256, compares signatures in constant time, and
// validates issuer, audience, and expiry.
package jwtutil

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	// DefaultIssuer and DefaultAudience identify tokens minted by the API.
	DefaultIssuer   = "fixphone-api"
	DefaultAudience = "fixphone-web"
)

var (
	// ErrExpired indicates a structurally valid token past its expiry.
	ErrExpired = errors.New("token kedaluwarsa")
	// ErrInvalid indicates any other verification failure.
	ErrInvalid = errors.New("token tidak valid")
)

// Claims is the access-token payload. PermissionVersion lets the server
// invalidate tokens immediately after a password or role change.
type Claims struct {
	Issuer            string `json:"iss"`
	Audience          string `json:"aud"`
	Subject           string `json:"sub"`
	TenantID          string `json:"tid,omitempty"`
	Role              string `json:"role"`
	SessionID         string `json:"sid"`
	PermissionVersion int    `json:"pv"`
	IssuedAt          int64  `json:"iat"`
	ExpiresAt         int64  `json:"exp"`
	ID                string `json:"jti"`
}

// Manager mints and verifies tokens with a fixed secret and lifetime.
type Manager struct {
	secret   []byte
	ttl      time.Duration
	issuer   string
	audience string
	now      func() time.Time
}

// NewManager builds a Manager. The secret must contain at least 32 bytes.
func NewManager(secret string, ttl time.Duration) (*Manager, error) {
	if len(secret) < 32 {
		return nil, errors.New("JWT_SECRET minimal 32 karakter")
	}
	if ttl <= 0 {
		return nil, errors.New("umur access token harus positif")
	}
	return &Manager{
		secret:   []byte(secret),
		ttl:      ttl,
		issuer:   DefaultIssuer,
		audience: DefaultAudience,
		now:      func() time.Time { return time.Now().UTC() },
	}, nil
}

// TTL returns the configured access-token lifetime.
func (m *Manager) TTL() time.Duration { return m.ttl }

// Issue creates a signed access token for the user and session.
func (m *Manager) Issue(userID uuid.UUID, tenantID *uuid.UUID, role, sessionID string, permissionVersion int) (string, time.Time, error) {
	now := m.now()
	expires := now.Add(m.ttl)
	jti, err := uuid.NewV7()
	if err != nil {
		return "", time.Time{}, err
	}
	if _, err := uuid.Parse(sessionID); err != nil {
		return "", time.Time{}, fmt.Errorf("session id tidak valid: %w", err)
	}

	claims := Claims{
		Issuer:            m.issuer,
		Audience:          m.audience,
		Subject:           userID.String(),
		Role:              role,
		SessionID:         sessionID,
		PermissionVersion: permissionVersion,
		IssuedAt:          now.Unix(),
		ExpiresAt:         expires.Unix(),
		ID:                jti.String(),
	}
	if tenantID != nil {
		claims.TenantID = tenantID.String()
	}

	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", time.Time{}, err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", time.Time{}, err
	}
	signingInput := base64.RawURLEncoding.EncodeToString(headerJSON) + "." +
		base64.RawURLEncoding.EncodeToString(claimsJSON)
	signature := m.sign(signingInput)
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature), expires, nil
}

// Parse verifies a token and returns its claims.
func (m *Manager) Parse(token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalid
	}

	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, ErrInvalid
	}
	var header struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, ErrInvalid
	}
	if header.Alg != "HS256" || header.Typ != "JWT" {
		return nil, ErrInvalid
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, ErrInvalid
	}
	if !hmac.Equal(signature, m.sign(parts[0]+"."+parts[1])) {
		return nil, ErrInvalid
	}

	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalid
	}
	var claims Claims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, ErrInvalid
	}

	now := m.now()
	if claims.ExpiresAt == 0 || now.Unix() >= claims.ExpiresAt {
		return nil, ErrExpired
	}
	if claims.IssuedAt == 0 || claims.IssuedAt > now.Add(time.Minute).Unix() {
		return nil, ErrInvalid
	}
	if claims.Issuer != m.issuer || claims.Audience != m.audience {
		return nil, ErrInvalid
	}
	if _, err := uuid.Parse(claims.Subject); err != nil {
		return nil, ErrInvalid
	}
	if _, err := uuid.Parse(claims.SessionID); err != nil {
		return nil, ErrInvalid
	}
	return &claims, nil
}

func (m *Manager) sign(input string) []byte {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(input))
	return mac.Sum(nil)
}
