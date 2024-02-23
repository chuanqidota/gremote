package response

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Success(c *gin.Context, msg string, data any) {
	c.JSON(http.StatusOK, gin.H{
		"msg":  msg,
		"code": 1,
		"data": data,
	})
}

func Fail(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, gin.H{
		"msg":  msg,
		"code": 0,
		"data": nil,
	})
}

// File 文件响应
func File(c *gin.Context, filename string, res []byte) {
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Data(http.StatusOK, "application/octet-stream", res)

}

func KeyRes(c *gin.Context, key string) {
	c.JSON(http.StatusOK, gin.H{
		"key": key,
	})
}
