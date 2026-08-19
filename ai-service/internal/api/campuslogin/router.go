package campuslogin

import (
	appModule "diaxel/internal/app"

	"github.com/gin-gonic/gin"
)

func CampusLoginRoutes(router *gin.Engine, app *appModule.App) {
	h := NewCampusLoginHandler(app.Cfg, app.Db, app.Twilio, app.LLM)

	group := router.Group("campuslogin")
	{
		group.POST("/triger-twilio/:assistant_id", h.HandleTriggerTwilio)
		group.POST("/triger-twilio/reinquiry/:assistant_id", h.HandleTriggerTwilioReinquiry)
		group.GET("/appointments/pending", h.GetPendingAppointments)
		group.PUT("/appointments/:id/sync", h.UpdateAppointmentSyncStatus)
	}
}
