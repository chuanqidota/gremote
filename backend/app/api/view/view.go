package view

import (
	"bytes"
	"fmt"
	"time"
	"gremote/app/api/params"
	"gremote/app/ws/utils/recordAudit"
	"gremote/config"
	"gremote/pkg/s3"
	"gremote/pkg/response"

	"strings"
	"gremote/pkg/redis"

	"gremote/pkg/file"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gremote/app/ws/utils/loginAudit"
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

	if err := redis.Set(key, info, time.Duration(config.Conf.Server.SessionTTL)*time.Second); err != nil {
		response.Fail(c, fmt.Sprintf("redis设置失败-%s", err.Error()))
		return
	}
	response.KeyRes(c, key)
}

// ObtainKeyRDP 获取RDP连接key
func (a *apiHandle) ObtainKeyRDP(c *gin.Context) {
	var info params.RDPInfo
	if err := c.ShouldBindJSON(&info); err != nil {
		response.Fail(c, fmt.Sprintf("参数错误-%s", err.Error()))
		return
	}
	if info.Port == 0 {
		info.Port = 3389
	}
	info.Source = c.ClientIP()
	key := uuid.New().String()
	if err := redis.Set(key, info, time.Duration(config.Conf.Server.SessionTTL)*time.Second); err != nil {
		response.Fail(c, fmt.Sprintf("redis设置失败-%s", err.Error()))
		return
	}
	response.KeyRes(c, key)
}

// ListFile 列出目录下的文件-大小-类型
func (a *apiHandle) ListFile(c *gin.Context) {
	// 获取key和path信息
	var listFileBody params.ListFileBody
	if err := c.ShouldBindQuery(&listFileBody); err != nil {
		response.Fail(c, fmt.Sprintf("传入参数错误-%s", err.Error()))
		return
	}

	// 通过key->target/username/password/port
	var info params.Info
	if err := redis.Get(listFileBody.Key, &info); err != nil {
		response.Fail(c, fmt.Sprintf("没有登录信息-%s", err.Error()))
		return
	}
	result, err := file.FileHandle.ListFile(info, listFileBody.Path)
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
	if err := c.ShouldBindQuery(&uploadFileBody); err != nil {
		response.Fail(c, fmt.Sprintf("参数错误-%s", err.Error()))
		return
	}

	// 通过key->target/username/password/port
	var info params.Info
	if err := redis.Get(uploadFileBody.Key, &info); err != nil {
		response.Fail(c, fmt.Sprintf("没有登录信息-%s", err.Error()))
		return
	}

	if err := file.FileHandle.UploadFile(fileObj, info, uploadFileBody.Path); err != nil {
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
	var info params.Info
	if err := redis.Get(downLoadFileBody.Key, &info); err != nil {
		response.Fail(c, fmt.Sprintf("没有登录信息-%s", err.Error()))
		return
	}

	filename := downLoadFileBody.FileName
	res, err := file.FileHandle.DownLoadFile(info, downLoadFileBody.Path, downLoadFileBody.FileName)
	if err != nil {
		response.Fail(c, fmt.Sprintf("执行失败-%s", err.Error()))
		return
	}
	response.File(c, filename, res)
}

// LoginAudit 登录审计查询
func (a *apiHandle) LoginAudit(c *gin.Context) {
	var data params.LoginAuditQuery
	if err := c.ShouldBindQuery(&data); err != nil {
		response.Fail(c, fmt.Sprintf("传参出错-%s", err.Error()))
		return
	}
	e := loginAudit.NewEsAudit()
	res, count := e.ReadData(data)
	result := map[string]any{
		"result": res,
		"count":  count,
	}
	response.Success(c, "执行成功", result)
}

// RecordUrl 获取记录的url
func (a *apiHandle) RecordUrl(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		response.Fail(c, "参数错误")
		return
	}
	// 从es中读取数据
	record := recordAudit.NewEsRecord()
	result := record.ReadData(key)

	var buffer bytes.Buffer
	for _, value := range result {
		history, _ := value["history"].(string)
		buffer.Write([]byte(history))
		buffer.WriteByte('\n')
	}

	// 上传到S3中-会覆盖更新
	if err := s3.UploadFile(key, buffer.Bytes()); err != nil {
		response.Fail(c, fmt.Sprintf("上传录制文件失败-%s", err.Error()))
		return
	}

	url := fmt.Sprintf("/api/v1/record-file?key=%s", key)
	response.Success(c, "执行成功", url)
}

// RecordFile 获取录制文件内容
func (a *apiHandle) RecordFile(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		response.Fail(c, "参数错误")
		return
	}
	data, err := s3.GetFile(key)
	if err != nil {
		response.Fail(c, fmt.Sprintf("读取录制文件失败-%s", err.Error()))
		return
	}
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Data(200, "text/plain; charset=utf-8", data)
}

// RecordFileGuac 获取RDP录制文件内容(.guac格式)
func (a *apiHandle) RecordFileGuac(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		response.Fail(c, "参数错误")
		return
	}
	guacKey := fmt.Sprintf("%s.guac", key)
	data, err := s3.GetFile(guacKey)
	if err != nil {
		response.Fail(c, fmt.Sprintf("读取RDP录制文件失败-%s", err.Error()))
		return
	}
	c.Header("Content-Type", "application/octet-stream")
	c.Data(200, "application/octet-stream", data)
}

// ListGuacFiles 列出S3中所有.guac录制文件（调试用）
func (a *apiHandle) ListGuacFiles(c *gin.Context) {
	files, err := s3.ListFiles("")
	if err != nil {
		response.Fail(c, fmt.Sprintf("列出文件失败-%s", err.Error()))
		return
	}
	var guacFiles []string
	for _, f := range files {
		if len(f) > 5 && f[len(f)-5:] == ".guac" {
			guacFiles = append(guacFiles, f)
		}
	}
	response.Success(c, "执行成功", guacFiles)
}

// GetConfig 获取前端显示配置
func (a *apiHandle) GetConfig(c *gin.Context) {
	c.JSON(200, gin.H{
		"display_mode": config.Conf.Display.DisplayMode,
	})
}
