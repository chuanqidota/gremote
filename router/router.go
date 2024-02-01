package router

import (
	"webssh-go/app/ws/view"

	"github.com/gin-gonic/gin"
)

func Engine() *gin.Engine {
	router := gin.Default()
	v1 := router.Group("v1")

	{
		v1.GET("ws/:key", view.WsHandle.Handler)
	}

	return router
}
