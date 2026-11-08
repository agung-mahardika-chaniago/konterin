package invoice

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts invoice endpoints.
func RegisterRoutes(api *gin.RouterGroup, h *Handler, auth, tenant gin.HandlerFunc, can func(string) gin.HandlerFunc) {
	group := api.Group("", auth, tenant)

	group.GET("/invoices", can("invoice.view"), h.ListInvoices)
	group.GET("/invoices/:id", can("invoice.view"), h.GetInvoice)
	group.PATCH("/invoices/:id", can("invoice.update"), h.UpdateInvoice)
	group.POST("/invoices/:id/issue", can("invoice.create"), h.IssueInvoice)
	group.POST("/invoices/:id/void", can("invoice.void"), h.VoidInvoice)
	group.GET("/invoices/:id/print", can("invoice.view"), h.PrintInvoice)
	group.GET("/service-orders/:id/invoice", can("invoice.view"), h.GetInvoiceByServiceOrder)
}
