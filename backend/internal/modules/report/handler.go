package report

import (
	"net/url"
	"strings"
	"time"

	"fixphone/backend/internal/middleware"
	"fixphone/backend/internal/pkg/apperror"
	"fixphone/backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler exposes report endpoints.
type Handler struct {
	service *Service
}

// NewHandler builds the report handler.
func NewHandler(service *Service) *Handler { return &Handler{service: service} }

// actorInfo carries the tenant scoping plus display name of the caller.
type actorInfo struct {
	TenantID   uuid.UUID
	OutletID   *uuid.UUID
	OutletIDs  []uuid.UUID
	ActorName  string
}

// actorFrom builds the actor scope; reports are tenant-only endpoints.
func actorFrom(c *gin.Context) (actorInfo, error) {
	scope, ok := middleware.GetScope(c)
	if !ok || scope.IsSuperAdmin || scope.TenantID == uuid.Nil {
		return actorInfo{}, apperror.Forbidden("FORBIDDEN", "Laporan hanya tersedia untuk pengguna tenant.")
	}
	auth, hasAuth := middleware.GetAuth(c)
	name := ""
	if hasAuth {
		name = auth.Name
	}
	return actorInfo{
		TenantID:  scope.TenantID,
		OutletID: scope.ActiveOutletID,
		ActorName: name,
	}, nil
}

// filterFrom reads the common report query parameters.
func filterFrom(c *gin.Context, actor actorInfo) (Filter, error) {
	filter := Filter{TenantID: actor.TenantID.String()}
	if raw := strings.TrimSpace(c.Query("outlet_id")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return filter, apperror.Validation(map[string]string{"outlet_id": "ID outlet tidak valid."})
		}
		scope, _ := middleware.GetScope(c)
		if scope != nil && !scope.HasOutlet(id) {
			return filter, apperror.Forbidden("OUTLET_FORBIDDEN", "Anda tidak memiliki akses ke outlet ini.")
		}
		filter.OutletID = id.String()
	} else if actor.OutletID != nil {
		filter.OutletID = actor.OutletID.String()
	}
	filter.From = strings.TrimSpace(c.Query("from"))
	filter.To = strings.TrimSpace(c.Query("to"))
	return filter, nil
}

// parsePeriod validates the period parameter with a default.
func parsePeriod(raw string) (Period, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "daily":
		return PeriodDaily, nil
	case "monthly":
		return PeriodMonthly, nil
	case "yearly":
		return PeriodYearly, nil
	default:
		return "", apperror.Validation(map[string]string{"period": "Periode harus daily, monthly, atau yearly."})
	}
}

// sendFile streams an xlsx export with download headers.
func sendFile(c *gin.Context, fileName string, data []byte) {
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(fileName))
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(200, "application/octet-stream", data)
}

// exportMeta builds the metadata block for exports.
func (h *Handler) exportMeta(c *gin.Context, actor actorInfo, sc Filter, report string) ExportMeta {
	return ExportMeta{
		Report: report, From: sc.From, To: sc.To,
		GeneratedBy: actor.ActorName,
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
	}
}

// Revenue returns revenue grouped by day/month/year.
func (h *Handler) Revenue(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	filter, err := filterFrom(c, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	period, err := parsePeriod(c.Query("period"))
	if err != nil {
		response.Error(c, err)
		return
	}

	result, err := h.service.Revenue(c.Request.Context(), filter, period)
	if err != nil {
		response.Error(c, err)
		return
	}
	if strings.ToLower(c.Query("format")) == "xlsx" {
		meta := h.exportMeta(c, actor, filter, "Laporan Pendapatan")
		store, outlet, nameErr := h.service.storeNames(c.Request.Context(), filter)
		if nameErr != nil {
			response.Error(c, nameErr)
			return
		}
		meta.StoreName, meta.OutletName = store, outlet
		file, wbErr := buildRevenueWorkbook(meta, result)
		if wbErr != nil {
			response.Error(c, apperror.Internal(wbErr))
			return
		}
		data, wbErr := workbookBytes(file)
		if wbErr != nil {
			response.Error(c, apperror.Internal(wbErr))
			return
		}
		sendFile(c, "laporan-pendapatan-"+result.From+"_"+result.To+".xlsx", data)
		return
	}
	response.Success(c, result)
}

// Services returns the service order report.
func (h *Handler) Services(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	filter, err := filterFrom(c, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	result, err := h.service.Services(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, err)
		return
	}
	if strings.ToLower(c.Query("format")) == "xlsx" {
		meta := h.exportMeta(c, actor, filter, "Laporan Servis")
		store, outlet, nameErr := h.service.storeNames(c.Request.Context(), filter)
		if nameErr != nil {
			response.Error(c, nameErr)
			return
		}
		meta.StoreName, meta.OutletName = store, outlet
		file, wbErr := buildServicesWorkbook(meta, result)
		if wbErr != nil {
			response.Error(c, apperror.Internal(wbErr))
			return
		}
		data, wbErr := workbookBytes(file)
		if wbErr != nil {
			response.Error(c, apperror.Internal(wbErr))
			return
		}
		sendFile(c, "laporan-servis-"+result.From+"_"+result.To+".xlsx", data)
		return
	}
	response.Success(c, result)
}

// Brands returns the device brand report.
func (h *Handler) Brands(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	filter, err := filterFrom(c, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	result, err := h.service.Brands(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, err)
		return
	}
	if strings.ToLower(c.Query("format")) == "xlsx" {
		meta := h.exportMeta(c, actor, filter, "Laporan Merek Perangkat")
		store, outlet, nameErr := h.service.storeNames(c.Request.Context(), filter)
		if nameErr != nil {
			response.Error(c, nameErr)
			return
		}
		meta.StoreName, meta.OutletName = store, outlet
		file, wbErr := buildBrandsWorkbook(meta, result)
		if wbErr != nil {
			response.Error(c, apperror.Internal(wbErr))
			return
		}
		data, wbErr := workbookBytes(file)
		if wbErr != nil {
			response.Error(c, apperror.Internal(wbErr))
			return
		}
		sendFile(c, "laporan-merek-"+result.From+"_"+result.To+".xlsx", data)
		return
	}
	response.Success(c, result)
}

// ServiceVsParts returns the labour vs parts comparison.
func (h *Handler) ServiceVsParts(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	filter, err := filterFrom(c, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	result, err := h.service.ServiceVsParts(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, err)
		return
	}
	if strings.ToLower(c.Query("format")) == "xlsx" {
		meta := h.exportMeta(c, actor, filter, "Laporan Jasa vs Sparepart")
		store, outlet, nameErr := h.service.storeNames(c.Request.Context(), filter)
		if nameErr != nil {
			response.Error(c, nameErr)
			return
		}
		meta.StoreName, meta.OutletName = store, outlet
		file, wbErr := buildServiceVsPartsWorkbook(meta, result)
		if wbErr != nil {
			response.Error(c, apperror.Internal(wbErr))
			return
		}
		data, wbErr := workbookBytes(file)
		if wbErr != nil {
			response.Error(c, apperror.Internal(wbErr))
			return
		}
		sendFile(c, "laporan-jasa-vs-part-"+result.From+"_"+result.To+".xlsx", data)
		return
	}
	response.Success(c, result)
}

// Profit returns gross/net profit; requires report.profit.view.
func (h *Handler) Profit(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	filter, err := filterFrom(c, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	result, err := h.service.Profit(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, err)
		return
	}
	if strings.ToLower(c.Query("format")) == "xlsx" {
		meta := h.exportMeta(c, actor, filter, "Laporan Laba")
		store, outlet, nameErr := h.service.storeNames(c.Request.Context(), filter)
		if nameErr != nil {
			response.Error(c, nameErr)
			return
		}
		meta.StoreName, meta.OutletName = store, outlet
		file, wbErr := buildProfitWorkbook(meta, result)
		if wbErr != nil {
			response.Error(c, apperror.Internal(wbErr))
			return
		}
		data, wbErr := workbookBytes(file)
		if wbErr != nil {
			response.Error(c, apperror.Internal(wbErr))
			return
		}
		sendFile(c, "laporan-laba-"+result.From+"_"+result.To+".xlsx", data)
		return
	}
	response.Success(c, result)
}

// Expenses returns the expense report; requires report.profit.view.
func (h *Handler) Expenses(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	filter, err := filterFrom(c, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	result, err := h.service.Expenses(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, err)
		return
	}
	if strings.ToLower(c.Query("format")) == "xlsx" {
		meta := h.exportMeta(c, actor, filter, "Laporan Pengeluaran")
		store, outlet, nameErr := h.service.storeNames(c.Request.Context(), filter)
		if nameErr != nil {
			response.Error(c, nameErr)
			return
		}
		meta.StoreName, meta.OutletName = store, outlet
		file, wbErr := buildExpensesWorkbook(meta, result)
		if wbErr != nil {
			response.Error(c, apperror.Internal(wbErr))
			return
		}
		data, wbErr := workbookBytes(file)
		if wbErr != nil {
			response.Error(c, apperror.Internal(wbErr))
			return
		}
		sendFile(c, "laporan-pengeluaran-"+result.From+"_"+result.To+".xlsx", data)
		return
	}
	response.Success(c, result)
}

// Technicians returns the technician productivity report.
func (h *Handler) Technicians(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	filter, err := filterFrom(c, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	result, err := h.service.Technicians(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, err)
		return
	}
	if strings.ToLower(c.Query("format")) == "xlsx" {
		meta := h.exportMeta(c, actor, filter, "Laporan Produktivitas Teknisi")
		store, outlet, nameErr := h.service.storeNames(c.Request.Context(), filter)
		if nameErr != nil {
			response.Error(c, nameErr)
			return
		}
		meta.StoreName, meta.OutletName = store, outlet
		file, wbErr := buildTechniciansWorkbook(meta, result)
		if wbErr != nil {
			response.Error(c, apperror.Internal(wbErr))
			return
		}
		data, wbErr := workbookBytes(file)
		if wbErr != nil {
			response.Error(c, apperror.Internal(wbErr))
			return
		}
		sendFile(c, "laporan-teknisi-"+result.From+"_"+result.To+".xlsx", data)
		return
	}
	response.Success(c, result)
}

// Stock returns the sparepart stock report.
func (h *Handler) Stock(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	filter, err := filterFrom(c, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	result, err := h.service.Stock(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, err)
		return
	}
	if strings.ToLower(c.Query("format")) == "xlsx" {
		meta := h.exportMeta(c, actor, filter, "Laporan Stok Sparepart")
		store, outlet, nameErr := h.service.storeNames(c.Request.Context(), filter)
		if nameErr != nil {
			response.Error(c, nameErr)
			return
		}
		meta.StoreName, meta.OutletName = store, outlet
		file, wbErr := buildStockWorkbook(meta, result)
		if wbErr != nil {
			response.Error(c, apperror.Internal(wbErr))
			return
		}
		data, wbErr := workbookBytes(file)
		if wbErr != nil {
			response.Error(c, apperror.Internal(wbErr))
			return
		}
		sendFile(c, "laporan-stok.xlsx", data)
		return
	}
	response.Success(c, result)
}

// InventoryValuation returns the inventory valuation; profit-aware.
func (h *Handler) InventoryValuation(c *gin.Context) {
	actor, err := actorFrom(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	filter, err := filterFrom(c, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	result, err := h.service.InventoryValuation(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}
