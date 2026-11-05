package auth

import (
	"net/http"
	"strings"

	"fixphone/backend/internal/middleware"
	"fixphone/backend/internal/pkg/apperror"
	"fixphone/backend/internal/pkg/response"
	"fixphone/backend/internal/pkg/validator"

	"github.com/gin-gonic/gin"
)

// Handler exposes the auth HTTP endpoints.
type Handler struct {
	service       *Service
	secureCookies bool
}

// NewHandler builds the auth handler.
func NewHandler(service *Service, secureCookies bool) *Handler {
	return &Handler{service: service, secureCookies: secureCookies}
}

// Login authenticates a user and sets the refresh cookie.
func (h *Handler) Login(c *gin.Context) {
	var body LoginRequest
	if err := validator.Bind(c, &body, map[string]string{
		"Email": "email", "Password": "password",
	}); err != nil {
		response.Error(c, err)
		return
	}

	session, err := h.service.Login(c.Request.Context(), LoginInput{
		Email: body.Email, Password: body.Password,
	}, requestMeta(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	h.setRefreshCookie(c, session)
	response.Success(c, sessionBody(session))
}

// Refresh rotates the refresh token and returns a fresh access token.
func (h *Handler) Refresh(c *gin.Context) {
	if !strings.HasPrefix(strings.ToLower(c.GetHeader("Content-Type")), "application/json") {
		response.Error(c, apperror.New(http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "Content-Type harus application/json."))
		return
	}
	cookie, err := c.Cookie(refreshCookieName)
	if err != nil {
		response.Error(c, apperror.Unauthorized("INVALID_REFRESH_TOKEN", "Sesi tidak ditemukan. Silakan login kembali."))
		return
	}
	session, err := h.service.Refresh(c.Request.Context(), cookie, requestMeta(c))
	if err != nil {
		h.clearRefreshCookie(c)
		response.Error(c, err)
		return
	}
	h.setRefreshCookie(c, session)
	response.Success(c, sessionBody(session))
}

// Logout revokes the current session and clears the cookie.
func (h *Handler) Logout(c *gin.Context) {
	identity, ok := middleware.GetAuth(c)
	if !ok {
		response.Error(c, apperror.Unauthorized("UNAUTHORIZED", "Sesi tidak ditemukan."))
		return
	}
	cookie, _ := c.Cookie(refreshCookieName)
	if err := h.service.Logout(c.Request.Context(), identity.UserID, identity.SessionID, cookie, requestMeta(c)); err != nil {
		response.Error(c, err)
		return
	}
	h.clearRefreshCookie(c)
	response.Success(c, gin.H{"message": "Anda telah keluar."})
}

// Me returns the current user's profile.
func (h *Handler) Me(c *gin.Context) {
	identity, ok := middleware.GetAuth(c)
	if !ok {
		response.Error(c, apperror.Unauthorized("UNAUTHORIZED", "Sesi tidak ditemukan."))
		return
	}
	profile, err := h.service.Me(c.Request.Context(), identity.UserID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, ProfileResponse{User: profile})
}

// UpdateProfile edits the current user's own profile.
func (h *Handler) UpdateProfile(c *gin.Context) {
	identity, ok := middleware.GetAuth(c)
	if !ok {
		response.Error(c, apperror.Unauthorized("UNAUTHORIZED", "Sesi tidak ditemukan."))
		return
	}
	var body UpdateProfileRequest
	if err := validator.Bind(c, &body, map[string]string{
		"Name": "name", "Phone": "phone", "AvatarPath": "avatar_path",
	}); err != nil {
		response.Error(c, err)
		return
	}
	profile, err := h.service.UpdateProfile(c.Request.Context(), identity.UserID, ProfileInput{
		Name: body.Name, Phone: body.Phone, AvatarPath: body.AvatarPath,
	}, requestMeta(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, ProfileResponse{User: profile})
}

// ChangePassword changes the current user's password and revokes sessions.
func (h *Handler) ChangePassword(c *gin.Context) {
	identity, ok := middleware.GetAuth(c)
	if !ok {
		response.Error(c, apperror.Unauthorized("UNAUTHORIZED", "Sesi tidak ditemukan."))
		return
	}
	var body ChangePasswordRequest
	if err := validator.Bind(c, &body, map[string]string{
		"CurrentPassword": "current_password", "NewPassword": "new_password",
	}); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.service.ChangePassword(c.Request.Context(), identity.UserID, ChangePasswordInput{
		CurrentPassword: body.CurrentPassword, NewPassword: body.NewPassword,
	}, requestMeta(c)); err != nil {
		response.Error(c, err)
		return
	}
	h.clearRefreshCookie(c)
	response.Success(c, gin.H{"message": "Password berhasil diubah. Silakan login kembali."})
}

func (h *Handler) setRefreshCookie(c *gin.Context, session *Session) {
	maxAge := int(session.RefreshTokenExpires.Sub(nowUTC()).Seconds())
	if maxAge < 1 {
		maxAge = 1
	}
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(refreshCookieName, session.RefreshToken, maxAge, "/api/v1/auth", "", h.secureCookies, true)
}

func (h *Handler) clearRefreshCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(refreshCookieName, "", -1, "/api/v1/auth", "", h.secureCookies, true)
}

func sessionBody(session *Session) gin.H {
	return gin.H{
		"access_token": session.AccessToken,
		"token_type":   "Bearer",
		"expires_in":   session.ExpiresIn,
		"user":         session.User,
	}
}

func requestMeta(c *gin.Context) Meta {
	return Meta{
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		RequestID: response.RequestID(c),
	}
}
