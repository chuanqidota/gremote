package router

import (
	api_view "gwebssh/app/api/view"
	ws_view "gwebssh/app/ws/view"
	"gwebssh/pkg/middleware"

	"github.com/gin-gonic/gin"
)

func Engine() *gin.Engine {
	router := gin.Default()

	api := router.Group("api/v1").Use(middleware.CORSMiddleware())
	{
		api.POST("obtain-key", api_view.ApiHandle.ObtainKey)
		api.GET("list-file", api_view.ApiHandle.ListFile)
		api.POST("upload-file", api_view.ApiHandle.UploadFile)
		api.GET("download-file", api_view.ApiHandle.DownLoadFile)
		api.GET("login-audit", api_view.ApiHandle.LoginAudit)
		api.GET("record-url", api_view.ApiHandle.RecordUrl)
		api.GET("record-file", api_view.ApiHandle.RecordFile)
	}

	ws := router.Group("ws/v1").Use(middleware.CORSMiddleware())
	{
		ws.GET(":key", ws_view.WsHandle.Handler)
	}

	return router
}
