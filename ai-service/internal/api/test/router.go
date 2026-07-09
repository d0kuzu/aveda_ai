package test

import (
	appModule "diaxel/internal/app"
	"diaxel/internal/modules/campuslogin"
	"log"

	"github.com/gin-gonic/gin"
)

func TestRoutes(router *gin.Engine, app *appModule.App) {
	h := NewTestHandler(app.GoogleCalendar, campuslogin.NewClient(app.Cfg.CampusLoginAPI))

	group := router.Group("test")
	{
		group.GET("/calendar", h.TestCalendar)
		group.POST("/calendar/event", h.TestCreateEvent)
		group.Any("/campuslogin/appointment", h.TestSendAppointment)
		
		// Эндпоинт для вывода всех данных запроса
		group.Any("/echo", func(c *gin.Context) {
			_ = c.Request.ParseMultipartForm(32 << 20)
			_ = c.Request.ParseForm()
			body, _ := c.GetRawData()

			log.Printf("====== ECHO REQUEST ======\n")
			log.Printf("Method: %s\n", c.Request.Method)
			log.Printf("URL: %s\n", c.Request.URL.String())
			log.Printf("Host: %s\n", c.Request.Host)
			log.Printf("RemoteAddr: %s\n", c.Request.RemoteAddr)
			log.Printf("ClientIP: %s\n", c.ClientIP())
			log.Printf("Headers: %+v\n", c.Request.Header)
			log.Printf("Query Params: %+v\n", c.Request.URL.Query())
			log.Printf("Form: %+v\n", c.Request.Form)
			log.Printf("PostForm: %+v\n", c.Request.PostForm)
			log.Printf("MultipartForm: %+v\n", c.Request.MultipartForm)
			log.Printf("ContentLength: %d\n", c.Request.ContentLength)
			log.Printf("Body: %s\n", string(body))
			log.Printf("==========================\n")

			c.Status(200)
		})
	}
}
