package router

import (
	api_view "webssh-go/app/api/view"
	ws_view "webssh-go/app/ws/view"
	"webssh-go/pkg/middleware"

	"github.com/gin-gonic/gin"
)

func Engine() *gin.Engine {
	router := gin.Default()
	ws := router.Group("ws").Use(middleware.CORSMiddleware())
	{
		ws.GET("v1/:key", ws_view.WsHandle.Handler)                 // websocket连接的服务
		ws.POST("v1/obtain-key", api_view.ApiHandle.ObtainKey)      // 通过连接信息获取key
		ws.GET("v1/list-file", api_view.ApiHandle.ListFile)         // 列出目标服务器上的文件
		ws.POST("v1/upload-file", api_view.ApiHandle.UploadFile)    // 上传文件到目标服务器
		ws.GET("v1/download-file", api_view.ApiHandle.DownLoadFile) // 从目标文件下载文件
		ws.GET("v1/login-audit", api_view.ApiHandle.LoginAudit)     // 登录审计
		ws.GET("v1/record-url", api_view.ApiHandle.RecordUrl)       // 获取记录的url
	}

	return router
}
