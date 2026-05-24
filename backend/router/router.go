package router

import (
	apiview "gwebssh/app/api/view"
	wsview "gwebssh/app/ws/view"
	"gwebssh/pkg/middleware"

	"github.com/gin-gonic/gin"
)

func Engine() *gin.Engine {
	router := gin.Default()

	api := router.Group("api/v1").Use(middleware.CORSMiddleware())
	{
		api.POST("obtain-key", apiview.ApiHandle.ObtainKey)
		api.POST("obtain-key-rdp", apiview.ApiHandle.ObtainKeyRDP)
		api.GET("list-file", apiview.ApiHandle.ListFile)
		api.POST("upload-file", apiview.ApiHandle.UploadFile)
		api.GET("download-file", apiview.ApiHandle.DownLoadFile)
		api.GET("login-audit", apiview.ApiHandle.LoginAudit)
		api.GET("record-url", apiview.ApiHandle.RecordUrl)
		api.GET("record-file", apiview.ApiHandle.RecordFile)
	}

	ws := router.Group("ws/v1").Use(middleware.CORSMiddleware())
	{
		ws.GET(":key", wsview.WsHandle.Handler)
		ws.GET("rdp/:key", wsview.WsHandle.RDPHandler)
	}

	return router
}
