package router

import (
	apiview "gremote/app/api/view"
	wsview "gremote/app/ws/view"
	"gremote/pkg/middleware"

	"github.com/gin-gonic/gin"
)

// Engine 创建并返回 Gin 路由引擎，注册所有 REST API 和 WebSocket 路由
func Engine() *gin.Engine {
	router := gin.Default()

	// REST API 路由组：会话管理、文件操作、审计查询、录制回放等
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
		api.GET("record-file-guac", apiview.ApiHandle.RecordFileGuac)
		api.GET("list-guac-files", apiview.ApiHandle.ListGuacFiles)
		api.POST("convert-guac", apiview.ApiHandle.ConvertGuac)
		api.GET("convert-status", apiview.ApiHandle.ConvertStatus)
		api.GET("record-file-mp4", apiview.ApiHandle.RecordFileMP4)
		api.GET("config", apiview.ApiHandle.GetConfig)
	}

	// WebSocket 路由组：SSH 终端和 RDP 远程桌面
	ws := router.Group("ws/v1").Use(middleware.CORSMiddleware())
	{
		ws.GET(":key", wsview.WsHandle.Handler)
		ws.GET("rdp/:key", wsview.WsHandle.RDPHandler)
	}

	return router
}
