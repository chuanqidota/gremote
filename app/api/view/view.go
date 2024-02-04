package view

import (
	"fmt"
	"time"
	"webssh-go/app/api/params"
	"webssh-go/pkg/response"

	"strings"
	"webssh-go/pkg/redis"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type apiHandle struct {
}

var ApiHandle = new(apiHandle)

// ObtainKey 获取key
func (a *apiHandle) ObtainKey(c *gin.Context) {
	var info params.Info
	if err := c.ShouldBindJSON(&info); err != nil {
		response.Fail(c, fmt.Sprintf("参数错误-%s", err.Error()))
		return
	}
	_uuid := uuid.New().String()
	key := strings.Replace(_uuid, "-", "", -1)

	if err := redis.Set(key, info, time.Second*60); err != nil {
		response.Fail(c, fmt.Sprintf("redis设置失败-%s", err.Error()))
		return
	}
	response.Success(c, "执行成功", map[string]string{"result": key})
}

// 目标目录下的文件
func (a *apiHandle) ListFile(c *gin.Context) {
	var listFileBody params.ListFileBody
	if err := c.ShouldBindJSON(&listFileBody); err != nil {
		response.Fail(c, fmt.Sprintf("传入参数错误-%s", err.Error()))
		return
	}
	var itemInfo params.ItemInfo

	if err := redis.Get(listFileBody.Key, &itemInfo); err != nil {
		response.Fail(c,fmt.Sprintf("没有登录信息-%s",err.Error()))
		return
	}



}

// UploadFile 上传文件
func (a *apiHandle) UploadFile(c *gin.Context) {

}

// DownLoadFile 下载文件
func (a *apiHandle) DownLoadFile(c *gin.Context) {

}
