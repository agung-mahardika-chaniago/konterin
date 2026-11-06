package service

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts customer and service-intake endpoints.
func RegisterRoutes(api *gin.RouterGroup, h *Handler, auth, tenant gin.HandlerFunc, can func(string) gin.HandlerFunc) {
	group := api.Group("", auth, tenant)

	group.GET("/customers", can("customer.view"), h.ListCustomers)
	group.POST("/customers", can("customer.create"), h.CreateCustomer)
	group.GET("/customers/:id", can("customer.view"), h.GetCustomer)
	group.PATCH("/customers/:id", can("customer.update"), h.UpdateCustomer)
	group.DELETE("/customers/:id", can("customer.delete"), h.DeleteCustomer)
	group.GET("/customers/:id/devices", can("customer.view"), h.ListCustomerDevices)
	group.POST("/customers/:id/devices", can("customer.update"), h.CreateCustomerDevice)
	group.GET("/customers/:id/services", can("customer.view"), h.ListCustomerServices)
	group.GET("/customers/:id/stats", can("customer.view"), h.CustomerStats)

	group.GET("/service-orders", can("service.view"), h.ListServiceOrders)
	group.POST("/service-orders", can("service.create"), h.CreateServiceOrder)
	group.GET("/service-orders/:id", can("service.view"), h.GetServiceOrder)
	group.PATCH("/service-orders/:id", can("service.update"), h.UpdateServiceOrder)
	group.PATCH("/service-orders/:id/status", can("service.status"), h.ChangeStatus)
	group.POST("/service-orders/:id/assign", can("service.assign"), h.AssignTechnician)
	group.GET("/service-orders/:id/timeline", can("service.view"), h.Timeline)

	group.GET("/service-orders/:id/diagnostic", can("diagnosis.view"), h.GetDiagnostic)
	group.PUT("/service-orders/:id/diagnostic", can("diagnosis.manage"), h.UpsertDiagnostic)

	group.GET("/service-orders/:id/items", can("service.view"), h.ListItems)
	group.POST("/service-orders/:id/items", can("service.update"), h.AddItem)
	group.PATCH("/service-orders/:id/items/:itemId", can("service.update"), h.UpdateItem)
	group.DELETE("/service-orders/:id/items/:itemId", can("service.update"), h.DeleteItem)

	group.GET("/spareparts/search", can("service.view"), h.SearchSpareparts)

	group.PUT("/service-orders/:id/credential", can("service.update"), h.SetCredential)
	group.DELETE("/service-orders/:id/credential", can("service.update"), h.DeleteCredential)
	group.POST("/service-orders/:id/credential/reveal", can("service.credential.reveal"), h.RevealCredential)

	group.POST("/service-orders/:id/photos", can("service.update"), h.UploadPhoto)
	group.DELETE("/service-orders/:id/photos/:photoId", can("service.update"), h.DeletePhoto)

	group.GET("/technicians", can("service.view"), h.Technicians)
}
