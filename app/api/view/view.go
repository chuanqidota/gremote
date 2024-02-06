package view

import (
	"fmt"
	"time"
	"webssh-go/app/api/params"
	"webssh-go/pkg/response"

	"strings"
	"webssh-go/pkg/redis"

	"webssh-go/pkg/file"

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

// ListFile 列出目录下的文件-大小-类型
func (a *apiHandle) ListFile(c *gin.Context) {
	// 获取key和path信息
	var listFileBody params.ListFileBody
	if err := c.ShouldBindJSON(&listFileBody); err != nil {
		response.Fail(c, fmt.Sprintf("传入参数错误-%s", err.Error()))
		return
	}

	// 通过key->target/username/password/port
	var itemInfo params.ItemInfo
	if err := redis.Get(listFileBody.Key, &itemInfo); err != nil {
		response.Fail(c, fmt.Sprintf("没有登录信息-%s", err.Error()))
		return
	}

	result, err := file.FileHandle.ListFile(itemInfo, listFileBody.Path)
	if err != nil {
		response.Fail(c, err.Error())
		return
	}
	response.Success(c, "执行成功", result)

}

// UploadFile 上传文件
func (a *apiHandle) UploadFile(c *gin.Context) {
	fileObj, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, fmt.Sprintf("上传文件失败-%s", err.Error()))
		return
	}

	var uploadFileBody params.UploadFileBody
	if err := c.ShouldBindJSON(&uploadFileBody); err != nil {
		response.Fail(c, fmt.Sprintf("参数错误-%s", err.Error()))
		return
	}

	// 通过key->target/username/password/port
	var itemInfo params.ItemInfo
	if err := redis.Get(uploadFileBody.Key, &itemInfo); err != nil {
		response.Fail(c, fmt.Sprintf("没有登录信息-%s", err.Error()))
		return
	}

	if err := file.FileHandle.UploadFile(fileObj, itemInfo, uploadFileBody.Path); err != nil {
		response.Fail(c, fmt.Sprintf("上传文件失败-%s", err.Error()))
		return
	}
	response.Success(c, "执行成功", nil)
}

// DownLoadFile 下载文件
func (a *apiHandle) DownLoadFile(c *gin.Context) {
	// 获取key和path信息
	var downLoadFileBody params.DownLoadFileBody
	if err := c.ShouldBindQuery(&downLoadFileBody); err != nil {
		response.Fail(c, fmt.Sprintf("传入参数错误-%s", err.Error()))
		return
	}

	// 通过key->target/username/password/port
	var itemInfo params.ItemInfo
	if err := redis.Get(downLoadFileBody.Key, &itemInfo); err != nil {
		response.Fail(c, fmt.Sprintf("没有登录信息-%s", err.Error()))
		return
	}

	filename := downLoadFileBody.FileName
	res, err := file.FileHandle.DownLoadFile(itemInfo, downLoadFileBody.Path, downLoadFileBody.FileName)
	if err != nil {
		response.Fail(c, fmt.Sprintf("执行失败-%s", err.Error()))
		return
	}
	response.File(c, filename, res)
}
