package inventory

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts sparepart master data and inventory endpoints.
func RegisterRoutes(api *gin.RouterGroup, h *Handler, auth, tenant gin.HandlerFunc, can func(string) gin.HandlerFunc) {
	group := api.Group("", auth, tenant)

	group.GET("/spareparts", can("sparepart.view"), h.ListSpareparts)
	group.POST("/spareparts", can("sparepart.create"), h.CreateSparepart)
	group.GET("/spareparts/barcode/:code", can("sparepart.view"), h.GetSparepartByBarcode)
	group.GET("/spareparts/:id", can("sparepart.view"), h.GetSparepart)
	group.PATCH("/spareparts/:id", can("sparepart.update"), h.UpdateSparepart)
	group.DELETE("/spareparts/:id", can("sparepart.delete"), h.DeleteSparepart)
	group.GET("/spareparts/:id/movements", can("inventory.view"), h.ListMovements)

	group.GET("/sparepart-categories", can("sparepart.view"), h.ListCategories)
	group.POST("/sparepart-categories", can("sparepart.create"), h.CreateCategory)
	group.PATCH("/sparepart-categories/:id", can("sparepart.update"), h.UpdateCategory)

	group.GET("/suppliers", can("sparepart.view"), h.ListSuppliers)
	group.POST("/suppliers", can("sparepart.create"), h.CreateSupplier)
	group.PATCH("/suppliers/:id", can("sparepart.update"), h.UpdateSupplier)

	group.POST("/inventory/purchases", can("inventory.purchase"), h.Purchase)
	group.POST("/inventory/adjustments", can("inventory.adjust"), h.AdjustStock)
	group.GET("/inventory/movements", can("inventory.view"), h.ListMovements)
	group.GET("/inventory/alerts", can("sparepart.view"), h.Alerts)
	group.GET("/inventory/valuation", can("inventory.cost.view"), h.Valuation)
	group.GET("/inventory/top-used", can("report.view"), h.TopUsed)
}
