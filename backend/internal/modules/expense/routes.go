package expense

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts expense endpoints.
func RegisterRoutes(api *gin.RouterGroup, h *Handler, auth, tenant gin.HandlerFunc, can func(string) gin.HandlerFunc) {
	group := api.Group("", auth, tenant)

	group.GET("/expenses", can("expense.view"), h.ListExpenses)
	group.POST("/expenses", can("expense.create"), h.CreateExpense)
	group.PATCH("/expenses/:id", can("expense.update"), h.UpdateExpense)
	group.DELETE("/expenses/:id", can("expense.delete"), h.DeleteExpense)
	group.GET("/expense-categories", can("expense.view"), h.ListCategories)
	group.POST("/expense-categories", can("expense.create"), h.CreateCategory)
}
