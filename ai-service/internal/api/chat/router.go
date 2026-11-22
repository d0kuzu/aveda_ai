package chat

import (
	appModule "diaxel/internal/app"

	"github.com/gin-gonic/gin"
)

func ChatRoutes(router *gin.Engine, app *appModule.App) {
	h := NewChatHandler(app.Cfg, app.Db)

	productGroup := router.Group("chats")
	{
		productGroup.GET("/get_all", h.GetAllChats)
		productGroup.GET("/get_chat", h.GetChat)
		productGroup.GET("/get_pagination", h.GetPagination)
		productGroup.GET("/search_chat", h.SearchChat)
	}
}
