package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"gremote/config"
	"gremote/pkg/logger"
	"gremote/pkg/response"
	"gremote/pkg/minio"

	"github.com/gin-gonic/gin"
)

var (
	converting sync.Map // key -> bool, tracks ongoing conversions
)

type workerRequest struct {
	Key string `json:"key"`
}

type workerResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// ConvertGuac triggers .guac to MP4 conversion (async)
func (a *apiHandle) ConvertGuac(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		response.Fail(c, "参数错误")
		return
	}

	// Check if already converting
	if _, loaded := converting.LoadOrStore(key, true); loaded {
		response.Success(c, "转换正在进行中", nil)
		return
	}

	// Check if MP4 already exists
	mp4Key := fmt.Sprintf("%s.mp4", key)
	if _, err := minio.GetFile(mp4Key); err == nil {
		converting.Delete(key)
		response.Success(c, "MP4已存在", nil)
		return
	}

	// Start async conversion
	go doConvert(key)
	response.Success(c, "转换已启动", nil)
}

func doConvert(key string) {
	defer converting.Delete(key)

	// Call worker
	workerURL := config.Conf.GuacWorker.URL
	timeout := time.Duration(config.Conf.GuacWorker.Timeout) * time.Second
	if timeout == 0 {
		timeout = 300 * time.Second
	}

	reqBody, _ := json.Marshal(workerRequest{Key: key})
	req, err := http.NewRequest("POST", workerURL+"/convert", bytes.NewReader(reqBody))
	if err != nil {
		logger.Error(fmt.Sprintf("转换失败-创建请求错误 key=%s err=%v", key, err))
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error(fmt.Sprintf("转换失败-调用worker错误 key=%s err=%v", key, err))
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error(fmt.Sprintf("转换失败-读取响应错误 key=%s err=%v", key, err))
		return
	}

	var workerResp workerResponse
	if err := json.Unmarshal(body, &workerResp); err != nil {
		logger.Error(fmt.Sprintf("转换失败-解析响应错误 key=%s err=%v", key, err))
		return
	}

	if !workerResp.Success {
		logger.Error(fmt.Sprintf("转换失败-worker返回错误 key=%s err=%s", key, workerResp.Error))
		return
	}

	logger.Info(fmt.Sprintf("转换完成 key=%s", key))
}

// ConvertStatus checks if MP4 conversion is complete
func (a *apiHandle) ConvertStatus(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		response.Fail(c, "参数错误")
		return
	}

	// Check if MP4 exists in S3
	mp4Key := fmt.Sprintf("%s.mp4", key)
	_, err := minio.GetFile(mp4Key)
	if err != nil {
		// Check if conversion is in progress
		_, converting := converting.Load(key)
		response.Success(c, "查询成功", gin.H{
			"converted":  false,
			"converting": converting,
		})
		return
	}

	response.Success(c, "查询成功", gin.H{
		"converted": true,
		"mp4_url":   fmt.Sprintf("/api/v1/record-file-mp4?key=%s", key),
	})
}

// RecordFileMP4 serves the converted MP4 recording
func (a *apiHandle) RecordFileMP4(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		response.Fail(c, "参数错误")
		return
	}
	mp4Key := fmt.Sprintf("%s.mp4", key)
	data, err := minio.GetFile(mp4Key)
	if err != nil {
		response.Fail(c, fmt.Sprintf("读取MP4录制文件失败-%s", err.Error()))
		return
	}
	c.Header("Content-Type", "video/mp4")
	c.Data(200, "video/mp4", data)
}
