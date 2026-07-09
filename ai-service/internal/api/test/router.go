package test

import (
	appModule "diaxel/internal/app"

	"github.com/gin-gonic/gin"
)

func TestRoutes(router *gin.Engine, app *appModule.App) {
	h := NewTestHandler(app.GoogleCalendar)

	group := router.Group("test")
	{
		group.GET("/calendar", h.TestCalendar)
		group.POST("/calendar/event", h.TestCreateEvent)
		
		// Эндпоинт для вывода всех данных запроса
		group.Any("/echo", func(c *gin.Context) {
			_ = c.Request.ParseMultipartForm(32 << 20)
			_ = c.Request.ParseForm()
			body, _ := c.GetRawData()

			c.JSON(200, gin.H{
				"method":         c.Request.Method,
				"url":            c.Request.URL.String(),
				"host":           c.Request.Host,
				"remote_addr":    c.Request.RemoteAddr,
				"headers":        c.Request.Header,
				"query_params":   c.Request.URL.Query(),
				"form":           c.Request.Form,
				"post_form":      c.Request.PostForm,
				"multipart_form": c.Request.MultipartForm,
				"client_ip":      c.ClientIP(),
				"content_length": c.Request.ContentLength,
				"body_raw":       string(body),
			})
		})
	}
}
