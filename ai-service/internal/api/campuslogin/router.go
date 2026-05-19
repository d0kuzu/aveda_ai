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
	}
}
