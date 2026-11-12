package report

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts report endpoints.
func RegisterRoutes(api *gin.RouterGroup, h *Handler, auth, tenant gin.HandlerFunc, can func(string) gin.HandlerFunc) {
	group := api.Group("/reports", auth, tenant)

	group.GET("/revenue", can("report.view"), h.Revenue)
	group.GET("/services", can("report.view"), h.Services)
	group.GET("/brands", can("report.view"), h.Brands)
	group.GET("/service-vs-parts", can("report.view"), h.ServiceVsParts)
	group.GET("/profit", can("report.profit.view"), h.Profit)
	group.GET("/expenses", can("report.profit.view"), h.Expenses)
	group.GET("/technicians", can("report.view"), h.Technicians)
	group.GET("/stock", can("report.view"), h.Stock)
	group.GET("/inventory-valuation", can("report.profit.view"), h.InventoryValuation)
}
