package router

import (
	apihandler "gremote/app/api/handler"
	wshandler "gremote/app/ws/handler"
	"gremote/pkg/middleware"

	"github.com/gin-gonic/gin"
)

// Engine 创建并返回 Gin 路由引擎，注册所有 REST API 和 WebSocket 路由
func Engine() *gin.Engine {
	router := gin.Default()

	// REST API 路由组：会话管理、文件操作、审计查询、录制回放等
	api := router.Group("api/v1").Use(middleware.CORSMiddleware())
	{
		api.POST("obtain-key", apihandler.ApiHandle.ObtainKey)
		api.POST("obtain-key-rdp", apihandler.ApiHandle.ObtainKeyRDP)
		api.GET("list-file", apihandler.ApiHandle.ListFile)
		api.POST("upload-file", apihandler.ApiHandle.UploadFile)
		api.GET("download-file", apihandler.ApiHandle.DownLoadFile)
		api.GET("login-audit", apihandler.ApiHandle.LoginAudit)
		api.GET("record-url", apihandler.ApiHandle.RecordUrl)
		api.GET("record-file", apihandler.ApiHandle.RecordFile)
		api.GET("record-file-guac", apihandler.ApiHandle.RecordFileGuac)
		api.GET("list-guac-files", apihandler.ApiHandle.ListGuacFiles)
		api.POST("convert-guac", apihandler.ApiHandle.ConvertGuac)
		api.GET("convert-status", apihandler.ApiHandle.ConvertStatus)
		api.GET("record-file-mp4", apihandler.ApiHandle.RecordFileMP4)
		api.GET("config", apihandler.ApiHandle.GetConfig)
	}

	// WebSocket 路由组：SSH 终端和 RDP 远程桌面
	ws := router.Group("ws/v1").Use(middleware.CORSMiddleware())
	{
		ws.GET(":key", wshandler.WsHandle.SSHHandler)
		ws.GET("rdp/:key", wshandler.WsHandle.RDPHandler)
	}

	return router
}
