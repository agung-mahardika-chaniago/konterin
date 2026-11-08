package payment

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts payment and cashier endpoints.
func RegisterRoutes(api *gin.RouterGroup, h *Handler, auth, tenant gin.HandlerFunc, can func(string) gin.HandlerFunc) {
	group := api.Group("", auth, tenant)

	group.GET("/invoices/:id/payments", can("payment.view"), h.ListInvoicePayments)
	group.POST("/invoices/:id/payments", can("payment.create"), h.CreatePayment)
	group.GET("/payments", can("payment.view"), h.ListPayments)
	group.GET("/payments/daily-summary", can("payment.view"), h.DailySummary)
	group.POST("/payments/:id/void", can("payment.void"), h.VoidPayment)
}
