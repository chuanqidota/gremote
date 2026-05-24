package view

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
	"gwebssh/app/api/params"
	"gwebssh/app/ws/utils/loginAudit"
	"gwebssh/config"
	"gwebssh/pkg/guacamole"
	"gwebssh/pkg/logger"
	"gwebssh/pkg/redis"
	"gwebssh/pkg/s3"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// guacInstruction is the JSON format exchanged between browser and backend.
type guacInstruction struct {
	Op   string   `json:"op"`
	Args []string `json:"args"`
}

// RDPHandler handles RDP WebSocket connections bridging browser to guacd.
func (w wsHandle) RDPHandler(c *gin.Context) {
	// 1. Upgrade HTTP to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Error(fmt.Sprintf("RDP WebSocket upgrade failed: %s", err.Error()))
		return
	}
	defer conn.Close()

	// 2. Validate key from Redis
	key := c.Param("key")
	if key == "" {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("无效链接"))
		return
	}
	if redis.IsConnected(key) {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("链接失效,已经被链接过一次"))
		return
	}

	// 3. Retrieve RDP info from Redis
	var info params.RDPInfo
	if err := redis.Get(key, &info); err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("获取登录信息失败"))
		return
	}

	// Auto-fill client IP
	clientIP := c.ClientIP()
	if info.User == "" {
		info.User = clientIP
	}
	if info.Source == "" {
		info.Source = clientIP
	}

	// 4. Write login audit to ES (with protocol: "rdp")
	e := loginAudit.NewEsAudit()
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

	// 5. Read first message for initial window size
	_, firstMessage, err := conn.ReadMessage()
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("读取初始消息失败"))
		return
	}
	var firstData map[string]any
	if err := json.Unmarshal(firstMessage, &firstData); err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("解析窗口大小失败"))
		return
	}
	widthFloat, ok1 := firstData["width"].(float64)
	heightFloat, ok2 := firstData["height"].(float64)
	if !ok1 || !ok2 {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("窗口大小数据格式错误"))
		return
	}
	width := int(widthFloat)
	height := int(heightFloat)

	// 6. Connect to guacd via Guacamole protocol client
	guacdHost := config.Conf.Guacd.Host
	guacdPort := config.Conf.Guacd.Port
	guacClient, err := guacamole.Connect(guacdHost, guacdPort)
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("连接guacd失败: %s", err.Error())))
		return
	}
	defer guacClient.Close()

	// 7. Perform Guacamole handshake (select rdp + params)
	guacParams := map[string]string{
		"hostname":        info.Target,
		"port":            fmt.Sprintf("%d", info.Port),
		"username":        info.Username,
		"password":        info.Password,
		"security":        "any",
		"ignore-cert":     "true",
		"disable-audio":   "true",
		"enable-wallpaper": "false",
		"enable-theming":  "false",
	}
	if info.Domain != "" {
		guacParams["domain"] = info.Domain
	}
	if err := guacClient.Handshake("rdp", guacParams); err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Guacamole握手失败: %s", err.Error())))
		return
	}

	// 8. Send initial size to guacd
	if err := guacClient.Write("size", fmt.Sprintf("%d", width), fmt.Sprintf("%d", height)); err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("发送初始尺寸失败: %s", err.Error())))
		return
	}

	// 9. Initialize recording (buffer data for MinIO upload)
	var recordingBuf bytes.Buffer
	var recordingMu sync.Mutex

	// 10. Start goroutines
	quitChan := make(chan bool, 2)
	var wsMu sync.Mutex // protects WebSocket writes

	// ReceiveWsMsg: WebSocket -> guacd
	go func() {
		defer func() { rdpSetQuit(quitChan) }()
		for {
			select {
			case <-quitChan:
				return
			default:
				_, message, err := conn.ReadMessage()
				if err != nil {
					logger.Error(fmt.Sprintf("RDP WS read error: %s", err.Error()))
					return
				}

				var instr guacInstruction
				if err := json.Unmarshal(message, &instr); err != nil {
					logger.Error(fmt.Sprintf("RDP WS message parse error: %s", err.Error()))
					continue
				}

				// Forward instruction to guacd
				if err := guacClient.Write(instr.Op, instr.Args...); err != nil {
					logger.Error(fmt.Sprintf("RDP guacd write error: %s", err.Error()))
					return
				}
			}
		}
	}()

	// WriteWsMsg: guacd -> WebSocket
	go func() {
		defer func() { rdpSetQuit(quitChan) }()
		for {
			select {
			case <-quitChan:
				return
			default:
				instr, err := guacClient.Read()
				if err != nil {
					logger.Error(fmt.Sprintf("RDP guacd read error: %s", err.Error()))
					return
				}

				// Convert to JSON for browser
				guacMsg := guacInstruction{
					Op:   instr.Op,
					Args: instr.Args,
				}
				jsonData, err := json.Marshal(guacMsg)
				if err != nil {
					logger.Error(fmt.Sprintf("RDP JSON marshal error: %s", err.Error()))
					continue
				}

				// Send to WebSocket
				wsMu.Lock()
				if err := conn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
					wsMu.Unlock()
					logger.Error(fmt.Sprintf("RDP WS write error: %s", err.Error()))
					return
				}
				wsMu.Unlock()

				// Buffer for recording (skip PNG data to save memory)
				if instr.Op != "img" {
					recordingMu.Lock()
					recordingBuf.Write([]byte(fmt.Sprintf("%s,%s;", instr.Op, joinArgs(instr.Args))))
					recordingMu.Unlock()
				}
			}
		}
	}()

	// 11. Wait for both goroutines to finish (with timeout)
	sessionDone := make(chan struct{})
	go func() {
		<-quitChan
		<-quitChan
		close(sessionDone)
	}()

	select {
	case <-sessionDone:
	case <-time.After(24 * time.Hour):
		logger.Error("RDP session timed out")
	}

	// Upload recording to MinIO
	recordingMu.Lock()
	if recordingBuf.Len() > 0 {
		recordingKey := fmt.Sprintf("%s.guac", key)
		for i := 0; i < 10; i++ {
			if err := s3.UploadFile(recordingKey, recordingBuf.Bytes()); err != nil {
				logger.Error(fmt.Sprintf("RDP recording upload failed: %s", err.Error()))
				time.Sleep(100 * time.Millisecond)
				continue
			}
			logger.Info(fmt.Sprintf("RDP recording uploaded: %s", recordingKey))
			break
		}
	}
	recordingMu.Unlock()

	logger.Info(fmt.Sprintf("RDP session ended for key: %s", key))
}

// joinArgs joins arguments with comma separator for recording format.
func joinArgs(args []string) string {
	return strings.Join(args, ",")
}

// rdpSetQuit signals the quit channel (non-blocking).
func rdpSetQuit(ch chan bool) {
	select {
	case ch <- true:
	default:
	}
}
