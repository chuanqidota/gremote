package router

import (
	api_view "webssh-go/app/api/view"
	ws_view "webssh-go/app/ws/view"

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
		api.POST("v1/obtain-key", api_view.ApiHandle.ObtainKey)       // 通过连接信息获取key
		api.POST("v1/list-file", api_view.ApiHandle.ListFile)         // 列出目标服务器上的文件
		api.POST("v1/upload-file", api_view.ApiHandle.UploadFile)     // 上传文件到目标服务器
		api.POST("v1/download-file", api_view.ApiHandle.DownLoadFile) // 从目标文件下载文件
	}

	return router
}
