package router

import (
	ws_view "webssh-go/app/ws/view"
	api_view "webssh-go/app/api/view"

	"github.com/gin-gonic/gin"
)

func Engine() *gin.Engine {
	router := gin.Default()
	ws := router.Group("ws")
	{
		ws.GET("v1/:key", ws_view.WsHandle.Handler)
	}

	api := router.Group("api")
	{
		api.POST("v1/obtain-key",api_view.ApiHandle.ObtainKey) // 获取key
	}

	return router
}
