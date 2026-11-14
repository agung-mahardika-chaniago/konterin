package settings

import (
	"net/url"
	"os"
	"strings"

	"fixphone/backend/internal/middleware"
	"fixphone/backend/internal/pkg/apperror"
	"fixphone/backend/internal/pkg/response"
	"fixphone/backend/internal/pkg/validator"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler exposes settings and backup endpoints.
type Handler struct {
	service *Service
}

// NewHandler builds the settings handler.
func NewHandler(service *Service) *Handler { return &Handler{service: service} }

// actorFrom builds the caller scope; settings is a tenant-only module.
func actorFrom(c *gin.Context) (Actor, error) {
	scope, ok := middleware.GetScope(c)
	if !ok || scope.IsSuperAdmin || scope.TenantID == uuid.Nil {
		return Actor{}, apperror.Forbidden("FORBIDDEN", "Pengaturan hanya tersedia untuk pengguna tenant.")
	}
	actor := Actor{TenantID: scope.TenantID, Name: ""}
	if auth, hasAuth := middleware.GetAuth(c); hasAuth {
		actor.UserID = auth.UserID
		actor.Name = auth.Name
	}
	return actor, nil
}

func uuidParam(c *gin.Context, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(c.Param(name)))
	if err != nil {
		return uuid.Nil, apperror.Validation(map[string]string{name: "ID tidak valid."})
	}
	return id, nil
}

// Get returns the tenant business settings.
func (h *Handler) Get(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	result, err := h.service.Get(c.Request.Context(), actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// Update partially updates business settings.
func (h *Handler) Update(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var body UpdateSettingsRequest
	if err := validator.Bind(c, &body, map[string]string{
		"store_name": "Nama toko", "legal_name": "Nama badan usaha",
		"address": "Alamat", "phone": "Telepon", "email": "Email", "tax_id": "NPWP",
		"timezone":      "Zona waktu",
		"ticket_prefix": "Prefiks tiket", "ticket_format": "Format tiket",
		"invoice_prefix": "Prefiks invoice", "invoice_format": "Format invoice",
		"invoice_terms": "Syarat invoice", "invoice_footer": "Catatan kaki invoice",
		"default_service_warranty_days": "Garansi jasa (hari)",
		"default_part_warranty_days":    "Garansi part (hari)",
		"receipt_paper_width":           "Lebar struk (mm)",
		"wa_provider":                   "Provider WhatsApp",
	}); err != nil {
		response.Error(c, err)
		return
	}
	result, err := h.service.Update(c.Request.Context(), actor, body)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// ListBackups returns the manual backup history.
func (h *Handler) ListBackups(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	items, err := h.service.ListBackups(c.Request.Context(), actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, items)
}

// CreateBackup dumps tenant data and records metadata.
func (h *Handler) CreateBackup(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	result, err := h.service.CreateBackup(c.Request.Context(), actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, result)
}

// DownloadBackup streams the backup file for the tenant.
func (h *Handler) DownloadBackup(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	backupID, err := uuidParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	path, err := h.service.GetBackupFile(c.Request.Context(), actor, backupID)
	if err != nil {
		response.Error(c, err)
		return
	}
	file, err := os.Open(path) // #nosec G304 -- path berasal dari metadata backup yang tersimpan, bukan input mentah pengguna.
	if err != nil {
		response.Error(c, apperror.Internal(err))
		return
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		response.Error(c, apperror.Internal(err))
		return
	}
	name := url.PathEscape(filepathBase(path))
	c.Header("Content-Type", "application/gzip")
	c.Header("Content-Disposition", "attachment; filename*=UTF-8''"+name)
	c.Header("X-Content-Type-Options", "nosniff")
	c.DataFromReader(200, info.Size(), "application/gzip", file, nil)
}

func filepathBase(path string) string {
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		return path[idx+1:]
	}
	return path
}
