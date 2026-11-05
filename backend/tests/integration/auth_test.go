package integration_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"fixphone/backend/internal/config"
	"fixphone/backend/internal/database"
	"fixphone/backend/internal/pkg/logger"
	"fixphone/backend/internal/router"
	"fixphone/backend/tests/testutil"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type authTestApp struct {
	engine   *gin.Engine
	db       *gorm.DB
	cfg      *config.Config
	tenantID uuid.UUID
	outletID uuid.UUID
	ownerID  uuid.UUID
}

type apiEnvelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Meta    json.RawMessage `json:"meta"`
	Error   struct {
		Code    string         `json:"code"`
		Message string         `json:"message"`
		Details map[string]any `json:"details"`
	} `json:"error"`
	RequestID string `json:"request_id"`
}

type apiResult struct {
	response *httptest.ResponseRecorder
	body     apiEnvelope
	cookies  []*http.Cookie
}

func setupAuthApp(t *testing.T) *authTestApp {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := freshDB(t)
	seedAuth(t, db)

	tenant, err := database.CreateTenant(t.Context(), db, database.CreateTenantInput{
		Name: "Toko Demo", Slug: "toko-demo", OwnerName: "Budi Owner",
		OwnerEmail: "owner@fixphone.local", OwnerPassword: "Owner#12345",
		OutletName: "Pusat", OutletCode: "PST",
	})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	cfg := &config.Config{
		AppEnv: "test", AppPort: "8080", DBDSN: testutil.DSN(),
		JWTSecret:         "integration-test-secret-that-is-long-enough",
		AccessTokenTTL:    15 * time.Minute,
		RefreshTokenTTL:   30 * 24 * time.Hour,
		LoginMaxAttempts:  5,
		LoginLockDuration: 15 * time.Minute,
		LoginRateLimit:    1000,
		LoginRateWindow:   time.Minute,
		CORSOrigins:       []string{"http://localhost:5173"},
		AppEncKey:         []byte("0123456789abcdef0123456789abcdef"),
		UploadDir:         t.TempDir(),
		UploadMaxMB:       5,
		AppBaseURL:        "http://127.0.0.1:5173",
	}
	log := logger.New(slog.LevelError, "test")
	return &authTestApp{
		engine: router.New(cfg, db, log), db: db, cfg: cfg,
		tenantID: tenant.TenantID, outletID: tenant.OutletID, ownerID: tenant.OwnerID,
	}
}

func seedAuth(t *testing.T, db *gorm.DB) {
	t.Helper()
	_, err := database.Seed(t.Context(), db, database.SeedOptions{
		SuperadminName:     "Super Admin",
		SuperadminEmail:    "superadmin@fixphone.local",
		SuperadminPassword: "Superadmin#2026",
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
}

func (a *authTestApp) request(t *testing.T, method, path string, body any, token string, cookies ...*http.Cookie) *apiResult {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:12345"
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for _, cookie := range cookies {
		if cookie != nil {
			req.AddCookie(cookie)
		}
	}
	rec := httptest.NewRecorder()
	a.engine.ServeHTTP(rec, req)
	result := &apiResult{response: rec, cookies: rec.Result().Cookies()}
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &result.body); err != nil {
			t.Fatalf("decode %s %s response (%d): %v\n%s", method, path, rec.Code, err, rec.Body.String())
		}
	}
	return result
}

func (a *authTestApp) login(t *testing.T, email, password string) *apiResult {
	t.Helper()
	return a.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": email, "password": password,
	}, "")
}

func (a *authTestApp) loginOwner(t *testing.T) (string, *http.Cookie) {
	t.Helper()
	result := a.login(t, "owner@fixphone.local", "Owner#12345")
	if result.response.Code != http.StatusOK {
		t.Fatalf("owner login status = %d: %s", result.response.Code, result.response.Body.String())
	}
	return accessToken(t, result), refreshCookie(t, result)
}

func accessToken(t *testing.T, result *apiResult) string {
	t.Helper()
	var data struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(result.body.Data, &data); err != nil {
		t.Fatalf("decode access token: %v", err)
	}
	if data.AccessToken == "" {
		t.Fatal("access token is empty")
	}
	return data.AccessToken
}

func refreshCookie(t *testing.T, result *apiResult) *http.Cookie {
	t.Helper()
	for _, cookie := range result.cookies {
		if cookie.Name == "fixphone_refresh" {
			return cookie
		}
	}
	t.Fatal("refresh cookie not set")
	return nil
}

func (a *authTestApp) createUser(t *testing.T, token, name, email, role string, outletIDs []string) map[string]any {
	t.Helper()
	result := a.request(t, http.MethodPost, "/api/v1/users", map[string]any{
		"name": name, "email": email, "role": role,
		"password": "Password#123", "outlet_ids": outletIDs,
	}, token)
	if result.response.Code != http.StatusCreated {
		t.Fatalf("create %s status = %d: %s", role, result.response.Code, result.response.Body.String())
	}
	var data map[string]any
	if err := json.Unmarshal(result.body.Data, &data); err != nil {
		t.Fatalf("decode created user: %v", err)
	}
	return data
}

func (a *authTestApp) allowLogin(t *testing.T, email string) {
	t.Helper()
	if err := a.db.Exec(`UPDATE users SET must_change_password = false WHERE email = ?`, email).Error; err != nil {
		t.Fatalf("clear must_change_password: %v", err)
	}
}

func TestAuthLoginSessionAndMe(t *testing.T) {
	app := setupAuthApp(t)
	result := app.login(t, "owner@fixphone.local", "Owner#12345")
	if result.response.Code != http.StatusOK {
		t.Fatalf("login status = %d: %s", result.response.Code, result.response.Body.String())
	}
	if !result.body.Success {
		t.Fatalf("login envelope unsuccessful: %s", result.response.Body.String())
	}
	cookie := refreshCookie(t, result)
	if !cookie.HttpOnly {
		t.Fatal("refresh cookie must be HttpOnly")
	}
	if cookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("refresh cookie SameSite = %v, want Strict", cookie.SameSite)
	}
	if cookie.Path != "/api/v1/auth" {
		t.Fatalf("refresh cookie path = %q", cookie.Path)
	}
	if cookie.MaxAge <= 0 {
		t.Fatalf("refresh cookie MaxAge = %d", cookie.MaxAge)
	}
	if bytes.Contains(result.response.Body.Bytes(), []byte("password_hash")) {
		t.Fatal("login response leaked a password hash")
	}

	var data struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		User        struct {
			Role        string   `json:"role"`
			Permissions []string `json:"permissions"`
			Outlets     []struct {
				ID string `json:"id"`
			} `json:"outlets"`
		} `json:"user"`
	}
	if err := json.Unmarshal(result.body.Data, &data); err != nil {
		t.Fatalf("decode login data: %v", err)
	}
	if data.ExpiresIn != 900 {
		t.Fatalf("expires_in = %d, want 900", data.ExpiresIn)
	}
	if data.User.Role != "owner" || len(data.User.Permissions) != 46 {
		t.Fatalf("owner role/permissions = %s/%d", data.User.Role, len(data.User.Permissions))
	}
	if len(data.User.Outlets) != 1 {
		t.Fatalf("owner outlets = %d, want 1", len(data.User.Outlets))
	}

	me := app.request(t, http.MethodGet, "/api/v1/auth/me", nil, data.AccessToken)
	if me.response.Code != http.StatusOK {
		t.Fatalf("me status = %d: %s", me.response.Code, me.response.Body.String())
	}
	if me.response.Header().Get("X-Request-ID") == "" {
		t.Fatal("me response missing X-Request-ID")
	}

	unauthorized := app.request(t, http.MethodGet, "/api/v1/auth/me", nil, "")
	if unauthorized.response.Code != http.StatusUnauthorized || unauthorized.body.Error.Code != "UNAUTHORIZED" {
		t.Fatalf("unauthorized me = %d/%s", unauthorized.response.Code, unauthorized.body.Error.Code)
	}
}

func TestAuthRefreshRotationAndReuseRevokesFamily(t *testing.T) {
	app := setupAuthApp(t)
	_, cookie1 := app.loginOwner(t)

	first := app.request(t, http.MethodPost, "/api/v1/auth/refresh", map[string]any{}, "", cookie1)
	if first.response.Code != http.StatusOK {
		t.Fatalf("first refresh = %d: %s", first.response.Code, first.response.Body.String())
	}
	cookie2 := refreshCookie(t, first)
	if cookie2.Value == cookie1.Value {
		t.Fatal("refresh token was not rotated")
	}

	reused := app.request(t, http.MethodPost, "/api/v1/auth/refresh", map[string]any{}, "", cookie1)
	if reused.response.Code != http.StatusUnauthorized || reused.body.Error.Code != "TOKEN_REUSED" {
		t.Fatalf("reused refresh = %d/%s", reused.response.Code, reused.body.Error.Code)
	}

	afterReuse := app.request(t, http.MethodPost, "/api/v1/auth/refresh", map[string]any{}, "", cookie2)
	if afterReuse.response.Code != http.StatusUnauthorized {
		t.Fatalf("rotated token still valid after family revoke: %d", afterReuse.response.Code)
	}

	var active int64
	if err := app.db.Raw(`SELECT count(*) FROM refresh_tokens WHERE revoked_at IS NULL`).Scan(&active).Error; err != nil {
		t.Fatalf("count active tokens: %v", err)
	}
	if active != 0 {
		t.Fatalf("active refresh tokens = %d after reuse detection", active)
	}

	var reusedEvents int64
	if err := app.db.Raw(`SELECT count(*) FROM audit_logs WHERE action = 'auth.token_reuse'`).Scan(&reusedEvents).Error; err != nil {
		t.Fatalf("count reuse audits: %v", err)
	}
	if reusedEvents < 1 {
		t.Fatalf("token_reuse audit rows = %d, want at least 1", reusedEvents)
	}
}

func TestAuthLogoutRevokesRefreshToken(t *testing.T) {
	app := setupAuthApp(t)
	token, cookie := app.loginOwner(t)

	logout := app.request(t, http.MethodPost, "/api/v1/auth/logout", map[string]any{}, token, cookie)
	if logout.response.Code != http.StatusOK {
		t.Fatalf("logout = %d: %s", logout.response.Code, logout.response.Body.String())
	}
	if cleared := refreshCookie(t, logout); cleared.Value != "" || cleared.MaxAge >= 0 {
		t.Fatalf("logout did not clear refresh cookie: %+v", cleared)
	}

	refresh := app.request(t, http.MethodPost, "/api/v1/auth/refresh", map[string]any{}, "", cookie)
	if refresh.response.Code != http.StatusUnauthorized {
		t.Fatalf("refresh after logout = %d, want 401", refresh.response.Code)
	}
}

func TestAuthLockoutBlocksCorrectPasswordAndIsAudited(t *testing.T) {
	app := setupAuthApp(t)
	for i := 0; i < 5; i++ {
		result := app.login(t, "owner@fixphone.local", "Wrong#Password1")
		if result.response.Code != http.StatusUnauthorized {
			t.Fatalf("failed login %d status = %d", i+1, result.response.Code)
		}
	}

	var state struct {
		Failed int        `gorm:"column:failed_login_count"`
		Locked *time.Time `gorm:"column:locked_until"`
	}
	if err := app.db.Raw(`SELECT failed_login_count, locked_until FROM users WHERE email = ?`,
		"owner@fixphone.local").Scan(&state).Error; err != nil {
		t.Fatalf("read login state: %v", err)
	}
	if state.Failed != 5 || state.Locked == nil || !state.Locked.After(time.Now()) {
		t.Fatalf("lockout state = %+v", state)
	}

	blocked := app.login(t, "owner@fixphone.local", "Owner#12345")
	if blocked.response.Code != 429 || blocked.body.Error.Code != "ACCOUNT_LOCKED" {
		t.Fatalf("locked login = %d/%s", blocked.response.Code, blocked.body.Error.Code)
	}

	var failedAudits, lockoutAudits int64
	app.db.Raw(`SELECT count(*) FROM audit_logs WHERE action = 'auth.login_failed'`).Scan(&failedAudits)
	app.db.Raw(`SELECT count(*) FROM audit_logs WHERE action = 'auth.lockout'`).Scan(&lockoutAudits)
	if failedAudits < 5 || lockoutAudits != 1 {
		t.Fatalf("audit counts failed=%d lockout=%d", failedAudits, lockoutAudits)
	}
}

func TestAuthRBACAndTenantIsolation(t *testing.T) {
	app := setupAuthApp(t)
	ownerToken, _ := app.loginOwner(t)

	admin := app.createUser(t, ownerToken, "Admin Satu", "admin1@fixphone.local", "admin", []string{app.outletID.String()})
	tech := app.createUser(t, ownerToken, "Teknisi Satu", "tech1@fixphone.local", "technician", []string{app.outletID.String()})
	app.allowLogin(t, "admin1@fixphone.local")
	app.allowLogin(t, "tech1@fixphone.local")

	techLogin := app.login(t, "tech1@fixphone.local", "Password#123")
	if techLogin.response.Code != http.StatusOK {
		t.Fatalf("technician login = %d: %s", techLogin.response.Code, techLogin.response.Body.String())
	}
	techToken := accessToken(t, techLogin)
	if result := app.request(t, http.MethodGet, "/api/v1/users", nil, techToken); result.response.Code != http.StatusForbidden {
		t.Fatalf("technician list users = %d, want 403", result.response.Code)
	}
	if result := app.request(t, http.MethodGet, "/api/v1/outlets", nil, techToken); result.response.Code != http.StatusOK {
		t.Fatalf("technician list outlets = %d, want 200", result.response.Code)
	}

	adminLogin := app.login(t, "admin1@fixphone.local", "Password#123")
	if adminLogin.response.Code != http.StatusOK {
		t.Fatalf("admin login = %d", adminLogin.response.Code)
	}
	adminToken := accessToken(t, adminLogin)
	if result := app.request(t, http.MethodGet, "/api/v1/users", nil, adminToken); result.response.Code != http.StatusOK {
		t.Fatalf("admin list users = %d: %s", result.response.Code, result.response.Body.String())
	}
	if result := app.request(t, http.MethodPost, "/api/v1/users", map[string]any{
		"name": "Owner Palsu", "email": "fake-owner@fixphone.local", "role": "owner", "password": "Password#123",
	}, adminToken); result.response.Code != http.StatusForbidden {
		t.Fatalf("admin create owner = %d, want 403", result.response.Code)
	}
	adminID := admin["id"].(string)
	if result := app.request(t, http.MethodPatch, "/api/v1/users/"+adminID, map[string]any{
		"is_active": false,
	}, adminToken); result.response.Code != http.StatusConflict {
		t.Fatalf("admin deactivate self = %d, want 409", result.response.Code)
	}
	ownerID := app.ownerID.String()
	if result := app.request(t, http.MethodPatch, "/api/v1/users/"+ownerID, map[string]any{
		"name": "Diubah Admin",
	}, adminToken); result.response.Code != http.StatusForbidden {
		t.Fatalf("admin update owner = %d, want 403", result.response.Code)
	}

	// Second tenant must never expose tenant A users.
	tenantB, err := database.CreateTenant(t.Context(), app.db, database.CreateTenantInput{
		Name: "Toko B", Slug: "toko-b", OwnerName: "Owner B",
		OwnerEmail: "ownerb@fixphone.local", OwnerPassword: "OwnerB#123",
		OutletName: "Cabang B", OutletCode: "B1",
	})
	if err != nil {
		t.Fatalf("create tenant B: %v", err)
	}
	loginB := app.login(t, "ownerb@fixphone.local", "OwnerB#123")
	if loginB.response.Code != http.StatusOK {
		t.Fatalf("tenant B login = %d: %s", loginB.response.Code, loginB.response.Body.String())
	}
	tokenB := accessToken(t, loginB)
	if result := app.request(t, http.MethodGet, "/api/v1/users/"+tech["id"].(string), nil, tokenB); result.response.Code != http.StatusNotFound {
		t.Fatalf("tenant B read tenant A user = %d, want 404", result.response.Code)
	}
	listB := app.request(t, http.MethodGet, "/api/v1/users", nil, tokenB)
	if listB.response.Code != http.StatusOK {
		t.Fatalf("tenant B list = %d", listB.response.Code)
	}
	var listData []map[string]any
	if err := json.Unmarshal(listB.body.Data, &listData); err != nil {
		t.Fatalf("decode tenant B list: %v", err)
	}
	if len(listData) != 1 || listData[0]["email"] != "ownerb@fixphone.local" {
		t.Fatalf("tenant B saw unexpected users: %+v", listData)
	}
	if result := app.request(t, http.MethodPatch, "/api/v1/outlets/"+app.outletID.String(), map[string]any{
		"name": "Hijack",
	}, tokenB); result.response.Code != http.StatusNotFound {
		t.Fatalf("tenant B update tenant A outlet = %d, want 404", result.response.Code)
	}
	if tenantB.TenantID == app.tenantID {
		t.Fatal("tenant IDs collided")
	}
}

func TestUserPasswordResetAndChangePasswordRevokeSessions(t *testing.T) {
	app := setupAuthApp(t)
	ownerToken, ownerCookie := app.loginOwner(t)
	created := app.createUser(t, ownerToken, "Kasir Satu", "cashier1@fixphone.local", "cashier", []string{app.outletID.String()})
	app.allowLogin(t, "cashier1@fixphone.local")

	cashierLogin := app.login(t, "cashier1@fixphone.local", "Password#123")
	if cashierLogin.response.Code != http.StatusOK {
		t.Fatalf("cashier login = %d", cashierLogin.response.Code)
	}
	cashierToken := accessToken(t, cashierLogin)

	reset := app.request(t, http.MethodPost, "/api/v1/users/"+created["id"].(string)+"/reset-password", map[string]any{}, ownerToken)
	if reset.response.Code != http.StatusOK {
		t.Fatalf("reset password = %d: %s", reset.response.Code, reset.response.Body.String())
	}
	var resetData struct {
		TemporaryPassword string `json:"temporary_password"`
		MustChange        bool   `json:"must_change_password"`
	}
	if err := json.Unmarshal(reset.body.Data, &resetData); err != nil {
		t.Fatalf("decode reset: %v", err)
	}
	if len(resetData.TemporaryPassword) < 8 || !resetData.MustChange {
		t.Fatalf("unexpected reset payload: %+v", resetData)
	}

	// Password reset bumps permission_version, invalidating the old access token.
	if result := app.request(t, http.MethodGet, "/api/v1/auth/me", nil, cashierToken); result.response.Code != http.StatusUnauthorized {
		t.Fatalf("old access token after reset = %d, want 401", result.response.Code)
	}
	oldPassword := app.login(t, "cashier1@fixphone.local", "Password#123")
	if oldPassword.response.Code != http.StatusUnauthorized {
		t.Fatalf("old password after reset = %d, want 401", oldPassword.response.Code)
	}
	newLogin := app.login(t, "cashier1@fixphone.local", resetData.TemporaryPassword)
	if newLogin.response.Code != http.StatusOK {
		t.Fatalf("temporary password login = %d: %s", newLogin.response.Code, newLogin.response.Body.String())
	}
	var newLoginData struct {
		User struct {
			MustChangePassword bool `json:"must_change_password"`
		} `json:"user"`
	}
	if err := json.Unmarshal(newLogin.body.Data, &newLoginData); err != nil {
		t.Fatalf("decode temp login: %v", err)
	}
	if !newLoginData.User.MustChangePassword {
		t.Fatal("temporary password session must require a password change")
	}

	// Owner changes their own password; the old session is revoked immediately.
	change := app.request(t, http.MethodPost, "/api/v1/auth/change-password", map[string]string{
		"current_password": "Owner#12345", "new_password": "OwnerBaru#456",
	}, ownerToken)
	if change.response.Code != http.StatusOK {
		t.Fatalf("change password = %d: %s", change.response.Code, change.response.Body.String())
	}
	if result := app.request(t, http.MethodGet, "/api/v1/auth/me", nil, ownerToken); result.response.Code != http.StatusUnauthorized {
		t.Fatalf("old access token after change = %d, want 401", result.response.Code)
	}
	if result := app.request(t, http.MethodPost, "/api/v1/auth/refresh", map[string]any{}, "", ownerCookie); result.response.Code != http.StatusUnauthorized {
		t.Fatalf("old refresh after change = %d, want 401", result.response.Code)
	}
	forced := app.login(t, "owner@fixphone.local", "OwnerBaru#456")
	if forced.response.Code != http.StatusOK {
		t.Fatalf("new password login = %d", forced.response.Code)
	}
}

func TestLoginRateLimit(t *testing.T) {
	app := setupAuthApp(t)
	app.cfg.LoginRateLimit = 2
	// Rebuild the router so the limiter reads the new configuration.
	app.engine = router.New(app.cfg, app.db, logger.New(slog.LevelError, "test"))

	first := app.login(t, "nobody@fixphone.local", "Wrong#Password1")
	second := app.login(t, "nobody@fixphone.local", "Wrong#Password1")
	third := app.login(t, "nobody@fixphone.local", "Wrong#Password1")
	if first.response.Code != http.StatusUnauthorized || second.response.Code != http.StatusUnauthorized {
		t.Fatalf("initial login statuses = %d/%d", first.response.Code, second.response.Code)
	}
	if third.response.Code != http.StatusTooManyRequests || third.body.Error.Code != "RATE_LIMITED" {
		t.Fatalf("third login = %d/%s, want 429/RATE_LIMITED", third.response.Code, third.body.Error.Code)
	}
}
