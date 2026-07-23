package blockedcustomers

import (
	appModule "diaxel/internal/app"

	"github.com/gin-gonic/gin"
)

func BlockedCustomersRoutes(router *gin.Engine, app *appModule.App) {
	h := NewBlockedCustomersHandler(app.Cfg, app.Db)

	group := router.Group("blocked-customers")
	{
		group.GET("/", h.GetAllBlockedCustomers)
		group.POST("/", h.BlockCustomer)
	}
}
