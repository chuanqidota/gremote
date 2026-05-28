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
		api.POST("obtain-key", apihandler.ApiHandle.ObtainKey)           // 获取 SSH 会话密钥
		api.POST("obtain-key-rdp", apihandler.ApiHandle.ObtainKeyRDP)    // 获取 RDP 会话密钥
		api.GET("list-file", apihandler.ApiHandle.ListFile)              // 浏览远程目录文件列表
		api.POST("upload-file", apihandler.ApiHandle.UploadFile)         // 上传文件到远程服务器
		api.GET("download-file", apihandler.ApiHandle.DownLoadFile)      // 从远程服务器下载文件
		api.GET("login-audit", apihandler.ApiHandle.LoginAudit)          // 查询登录审计记录
		api.GET("record-url", apihandler.ApiHandle.RecordUrl)            // 获取 SSH 录制回放地址
		api.GET("record-file", apihandler.ApiHandle.RecordFile)          // 获取 SSH 录制文件内容(asciinema)
		api.GET("record-file-guac", apihandler.ApiHandle.RecordFileGuac) // 获取 RDP 录制文件内容(.guac)
		api.GET("list-guac-files", apihandler.ApiHandle.ListGuacFiles)   // 列出 S3 中所有 .guac 录制文件
		api.POST("convert-guac", apihandler.ApiHandle.ConvertGuac)       // 触发 .guac 转 MP4 异步任务
		api.GET("convert-status", apihandler.ApiHandle.ConvertStatus)    // 查询 MP4 转换状态
		api.GET("record-file-mp4", apihandler.ApiHandle.RecordFileMP4)   // 获取转换后的 MP4 录制文件
		api.GET("record-file-size", apihandler.ApiHandle.RecordFileSize) // 获取.guac文件大小及是否需要转换
		api.GET("config", apihandler.ApiHandle.GetConfig)                // 获取前端显示配置(display_mode)
	}

	// WebSocket 路由组：SSH 终端和 RDP 远程桌面
	ws := router.Group("ws/v1").Use(middleware.CORSMiddleware())
	{
		ws.GET(":key", wshandler.WsHandle.SSHHandler)     // SSH 终端 WebSocket 连接
		ws.GET("rdp/:key", wshandler.WsHandle.RDPHandler) // RDP 远程桌面 WebSocket 连接
	}

	return router
}
