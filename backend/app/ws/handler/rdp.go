package handler

import (
	"fmt"
	"os"
	"sync"
	"time"
	"gremote/app/api/params"
	"gremote/app/audit/loginAudit"
	"gremote/config"
	"gremote/pkg/guacamole"
	"gremote/pkg/logger"
	"gremote/pkg/redis"
	"gremote/pkg/minio"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// RDPHandler 处理 RDP WebSocket 连接，桥接浏览器与 guacd
func (w wsHandle) RDPHandler(c *gin.Context) {
	logger.Info(fmt.Sprintf("RDP WebSocket 连接请求来自 %s", c.ClientIP()))

	// 1. 升级 HTTP 为 WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Error(fmt.Sprintf("RDP WebSocket 升级失败: %s", err.Error()))
		return
	}
	defer conn.Close()

	logger.Info("RDP WebSocket 升级成功")

	// 2. 验证 Redis 中的会话密钥
	key := c.Param("key")
	if key == "" {
		logger.Error("RDP 空密钥")
		_ = conn.WriteMessage(websocket.TextMessage, []byte("无效链接"))
		return
	}
	if redis.IsConnected(key) {
		logger.Error(fmt.Sprintf("RDP 密钥已被使用: %s", key))
		_ = conn.WriteMessage(websocket.TextMessage, []byte("链接失效,已经被链接过一次"))
		return
	}

	// 3. 从 Redis 获取 RDP 连接信息
	var info params.RDPInfo
	if err := redis.Get(key, &info); err != nil {
		logger.Error(fmt.Sprintf("RDP Redis 获取失败 key=%s: %s", key, err.Error()))
		_ = conn.WriteMessage(websocket.TextMessage, []byte("获取登录信息失败"))
		return
	}

	logger.Info(fmt.Sprintf("RDP 连接信息: target=%s port=%d user=%s", info.Target, info.Port, info.Username))

	// 自动填充客户端 IP
	clientIP := c.ClientIP()
	if info.User == "" {
		info.User = clientIP
	}
	if info.Source == "" {
		info.Source = clientIP
	}

	// 4. 写入登录审计记录（protocol: "rdp"）
	e := loginAudit.NewLoginAudit()
	defer redis.DeleteKey(key)
	auditData := map[string]any{
		"key":       key,
		"startTime": time.Now().Format("2006-01-02 15:04:05"),
		"user":      info.User,
		"source":    info.Source,
		"target":    info.Target,
		"protocol":  "rdp",
	}
	e.WriteData(auditData)
	defer e.UpdateEndTime(key)

	// 5. 通过 Guacamole 协议连接 guacd
	guacdHost := config.Conf.Guacd.Host
	guacdPort := config.Conf.Guacd.Port
	logger.Info(fmt.Sprintf("RDP 正在连接 guacd %s:%d", guacdHost, guacdPort))
	guacClient, err := guacamole.Connect(guacdHost, guacdPort)
	if err != nil {
		logger.Error(fmt.Sprintf("RDP guacd 连接失败: %s", err.Error()))
		_ = conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("连接guacd失败: %s", err.Error())))
		return
	}
	logger.Info("RDP guacd 连接成功")
	// 注意：guacClient.Close() 会在读取录制文件前显式调用，确保 guacd 将录制数据刷写到磁盘

	// 6. 从查询参数获取视口大小（由浏览器发送）
	width := c.Query("width")
	height := c.Query("height")
	if width == "" {
		width = fmt.Sprintf("%d", config.Conf.Guacd.DefaultWidth)
	}
	if height == "" {
		height = fmt.Sprintf("%d", config.Conf.Guacd.DefaultHeight)
	}
	logger.Info(fmt.Sprintf("RDP 视口大小: %sx%s", width, height))

	// 7. 执行 Guacamole 握手（select rdp + 参数）
	guacParams := map[string]string{
		"hostname":              info.Target,
		"port":                  fmt.Sprintf("%d", info.Port),
		"username":              info.Username,
		"password":              info.Password,
		"width":                 width,
		"height":                height,
		"dpi":                   fmt.Sprintf("%d", config.Conf.Guacd.DefaultDPI),
		"security":              "any",
		"ignore-cert":           "true",
		"disable-audio":         "true",
		"enable-wallpaper":      "false",
		"enable-theming":        "false",
		"recording-path":        fmt.Sprintf("%s/%s.guac", config.Conf.Guacd.GuacdPath, key),
		"create-recording-path": "true",
	}
	if info.Domain != "" {
		guacParams["domain"] = info.Domain
	}
	logger.Info("RDP 开始 Guacamole 握手")
	if err := guacClient.Handshake("rdp", guacParams); err != nil {
		logger.Error(fmt.Sprintf("RDP Guacamole 握手失败: %s", err.Error()))
		_ = conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Guacamole握手失败: %s", err.Error())))
		return
	}
	logger.Info("RDP Guacamole 握手成功")

	// 8. 启动桥接协程
	quitChan := make(chan bool, 2)
	var wsMu sync.Mutex // 保护 WebSocket 写入

	logger.Info("RDP 启动桥接协程")

	// ReceiveWsMsg: WebSocket -> guacd（原始 Guacamole 协议）
	go func() {
		defer func() { rdpSetQuit(quitChan) }()
		for {
			select {
			case <-quitChan:
				return
			default:
				_, message, err := conn.ReadMessage()
				if err != nil {
					logger.Error(fmt.Sprintf("RDP WS 读取错误: %s", err.Error()))
					guacClient.Close()
					return
				}

				instr, err := guacamole.ParseInstruction(string(message))
				if err != nil {
					logger.Error(fmt.Sprintf("RDP WS 消息解析错误: %s", err.Error()))
					continue
				}

				if err := guacClient.Write(instr.Op, instr.Args...); err != nil {
					logger.Error(fmt.Sprintf("RDP guacd 写入错误: %s", err.Error()))
					return
				}
			}
		}
	}()

	// WriteWsMsg: guacd -> WebSocket（原始 Guacamole 协议）
	// 使用 10s 读取超时发送 nop 心跳，防止浏览器隧道在 guacd 建立 RDP 连接期间超时
	go func() {
		defer func() { rdpSetQuit(quitChan) }()
		for {
			select {
			case <-quitChan:
				return
			default:
			}

			instr, err := guacClient.ReadDeadline(10 * time.Second)
			if err != nil {
				if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
					wsMu.Lock()
					_ = conn.WriteMessage(websocket.TextMessage, []byte("3.nop;"))
					wsMu.Unlock()
					continue
				}
				logger.Error(fmt.Sprintf("RDP guacd 读取错误: %s", err.Error()))
				return
			}

			// 仅记录非流式指令，避免大量 blob 数据刷屏日志
			if instr.Op != "blob" && instr.Op != "end" {
				logger.Info(fmt.Sprintf("RDP guacd -> WS: op=%s args=%v", instr.Op, instr.Args))
			}

			rawInstr := guacamole.EncodeInstruction(instr.Op, instr.Args...)

			wsMu.Lock()
			if err := conn.WriteMessage(websocket.TextMessage, []byte(rawInstr)); err != nil {
				wsMu.Unlock()
				logger.Error(fmt.Sprintf("RDP WS 写入错误: %s", err.Error()))
				guacClient.Close()
				return
			}
			wsMu.Unlock()
		}
	}()

	// 9. 等待两个协程结束（带超时）
	sessionDone := make(chan struct{})
	go func() {
		<-quitChan
		<-quitChan
		close(sessionDone)
	}()

	select {
	case <-sessionDone:
	case <-time.After(time.Duration(config.Conf.Guacd.SessionTimeout) * time.Second):
		logger.Error("RDP 会话超时")
	}

	// 10. 上传 guacd 录制文件到 MinIO
	// 先关闭 guacd 连接，确保其将录制数据刷写到磁盘
	logger.Info("RDP 关闭 guacd 连接以刷写录制文件")
	guacClient.Close()

	// guacd 会创建目录 <key>.guac/，内含 "recording" 文件
	recordBasePath := config.Conf.Guacd.RecordingPath
	recordingPath := fmt.Sprintf("%s/%s.guac/recording", recordBasePath, key)
	recordingKey := fmt.Sprintf("%s.guac", key)
	logger.Info(fmt.Sprintf("RDP 查找录制文件: %s", recordingPath))

	var data []byte
	for i := 0; i < 30; i++ {
		time.Sleep(1 * time.Second)
		var err error
		data, err = os.ReadFile(recordingPath)
		if err == nil && len(data) > 0 {
			logger.Info(fmt.Sprintf("RDP 录制文件读取成功: %s (%d bytes)", recordingPath, len(data)))
			break
		}
		if err != nil {
			logger.Info(fmt.Sprintf("RDP 录制文件读取尝试 %d/30 失败: %s", i+1, err.Error()))
		} else {
			logger.Info(fmt.Sprintf("RDP 录制文件为空 尝试 %d/30", i+1))
		}
	}

	if len(data) > 0 {
		uploaded := false
		for i := 0; i < 10; i++ {
			if err := minio.UploadFile(recordingKey, data); err != nil {
				logger.Error(fmt.Sprintf("RDP S3 上传尝试 %d 失败: %s", i+1, err.Error()))
				time.Sleep(500 * time.Millisecond)
				continue
			}
			logger.Info(fmt.Sprintf("RDP 录制文件已上传到 S3: %s (%d bytes)", recordingKey, len(data)))
			uploaded = true
			break
		}
		if !uploaded {
			logger.Error(fmt.Sprintf("RDP S3 上传失败（重试10次）: %s", recordingKey))
		}
		// 清理本地录制目录
		dirPath := fmt.Sprintf("%s/%s.guac", recordBasePath, key)
		if err := os.RemoveAll(dirPath); err != nil {
			logger.Error(fmt.Sprintf("RDP 录制文件清理失败: %s", err.Error()))
		} else {
			logger.Info(fmt.Sprintf("RDP 录制文件清理成功: %s", dirPath))
		}
	} else {
		logger.Error(fmt.Sprintf("RDP 录制文件未找到（30次重试/30秒）: %s", recordingPath))
	}

	logger.Info(fmt.Sprintf("RDP 会话结束 key=%s", key))
}

// rdpSetQuit 向退出通道发送信号（非阻塞）
func rdpSetQuit(ch chan bool) {
	select {
	case ch <- true:
	default:
	}
}
