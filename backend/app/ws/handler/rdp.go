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

// RDPHandler handles RDP WebSocket connections bridging browser to guacd.
func (w wsHandle) RDPHandler(c *gin.Context) {
	logger.Info(fmt.Sprintf("RDP WebSocket connection attempt from %s", c.ClientIP()))

	// 1. Upgrade HTTP to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Error(fmt.Sprintf("RDP WebSocket upgrade failed: %s", err.Error()))
		return
	}
	defer conn.Close()

	logger.Info("RDP WebSocket upgraded successfully")

	// 2. Validate key from Redis
	key := c.Param("key")
	if key == "" {
		logger.Error("RDP empty key")
		_ = conn.WriteMessage(websocket.TextMessage, []byte("无效链接"))
		return
	}
	if redis.IsConnected(key) {
		logger.Error(fmt.Sprintf("RDP key already used: %s", key))
		_ = conn.WriteMessage(websocket.TextMessage, []byte("链接失效,已经被链接过一次"))
		return
	}

	// 3. Retrieve RDP info from Redis
	var info params.RDPInfo
	if err := redis.Get(key, &info); err != nil {
		logger.Error(fmt.Sprintf("RDP Redis get failed for key %s: %s", key, err.Error()))
		_ = conn.WriteMessage(websocket.TextMessage, []byte("获取登录信息失败"))
		return
	}

	logger.Info(fmt.Sprintf("RDP connection info: target=%s port=%d user=%s", info.Target, info.Port, info.Username))

	// Auto-fill client IP
	clientIP := c.ClientIP()
	if info.User == "" {
		info.User = clientIP
	}
	if info.Source == "" {
		info.Source = clientIP
	}

	// 4. Write login audit to ES (with protocol: "rdp")
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

	// 5. Connect to guacd via Guacamole protocol client
	guacdHost := config.Conf.Guacd.Host
	guacdPort := config.Conf.Guacd.Port
	logger.Info(fmt.Sprintf("RDP connecting to guacd at %s:%d", guacdHost, guacdPort))
	guacClient, err := guacamole.Connect(guacdHost, guacdPort)
	if err != nil {
		logger.Error(fmt.Sprintf("RDP guacd connect failed: %s", err.Error()))
		_ = conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("连接guacd失败: %s", err.Error())))
		return
	}
	logger.Info("RDP connected to guacd successfully")
	// NOTE: guacClient.Close() is called explicitly before reading the
	// recording file to ensure guacd flushes the recording to disk.

	// 6. Read viewport size from query params (sent by browser)
	width := c.Query("width")
	height := c.Query("height")
	if width == "" {
		width = fmt.Sprintf("%d", config.Conf.Guacd.DefaultWidth)
	}
	if height == "" {
		height = fmt.Sprintf("%d", config.Conf.Guacd.DefaultHeight)
	}
	logger.Info(fmt.Sprintf("RDP viewport size: %sx%s", width, height))

	// 7. Perform Guacamole handshake (select rdp + params)
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
	logger.Info("RDP starting Guacamole handshake")
	if err := guacClient.Handshake("rdp", guacParams); err != nil {
		logger.Error(fmt.Sprintf("RDP Guacamole handshake failed: %s", err.Error()))
		_ = conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Guacamole握手失败: %s", err.Error())))
		return
	}
	logger.Info("RDP Guacamole handshake completed successfully")

	// 7. Start bridge goroutines
	quitChan := make(chan bool, 2)
	var wsMu sync.Mutex // protects WebSocket writes

	logger.Info("RDP starting bridge goroutines")

	// ReceiveWsMsg: WebSocket -> guacd (raw Guacamole protocol)
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
					// WebSocket disconnected, close guacd to unblock WriteWsMsg
					guacClient.Close()
					return
				}

				// Parse raw Guacamole protocol instruction
				instr, err := guacamole.ParseInstruction(string(message))
				if err != nil {
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

	// WriteWsMsg: guacd -> WebSocket (raw Guacamole protocol)
	// Uses 10s read deadline to send periodic nop keepalives while guacd
	// establishes the RDP connection, preventing browser tunnel timeout.
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
				logger.Error(fmt.Sprintf("RDP guacd read error: %s", err.Error()))
				return
			}

			// Only log non-streaming instructions to avoid log spam from large blob data
			if instr.Op != "blob" && instr.Op != "end" {
				logger.Info(fmt.Sprintf("RDP guacd -> WS: op=%s args=%v", instr.Op, instr.Args))
			}

			// Encode instruction to raw Guacamole protocol
			rawInstr := guacamole.EncodeInstruction(instr.Op, instr.Args...)

			// Send to WebSocket
			wsMu.Lock()
			if err := conn.WriteMessage(websocket.TextMessage, []byte(rawInstr)); err != nil {
				wsMu.Unlock()
				logger.Error(fmt.Sprintf("RDP WS write error: %s", err.Error()))
				// WebSocket disconnected, close guacd to unblock ReceiveWsMsg
				guacClient.Close()
				return
			}
			wsMu.Unlock()
		}
	}()

	// 8. Wait for both goroutines to finish (with timeout)
	sessionDone := make(chan struct{})
	go func() {
		<-quitChan
		<-quitChan
		close(sessionDone)
	}()

	select {
	case <-sessionDone:
	case <-time.After(time.Duration(config.Conf.Guacd.SessionTimeout) * time.Second):
		logger.Error("RDP session timed out")
	}

	// 9. Upload guacd recording to MinIO
	// Close guacd connection first to ensure it flushes the recording to disk
	logger.Info("RDP closing guacd connection to flush recording")
	guacClient.Close()

	// guacd creates a directory <key>.guac/ with a "recording" file inside
	recordBasePath := config.Conf.Guacd.RecordingPath
	recordingPath := fmt.Sprintf("%s/%s.guac/recording", recordBasePath, key)
	recordingKey := fmt.Sprintf("%s.guac", key)
	logger.Info(fmt.Sprintf("RDP looking for recording at: %s", recordingPath))

	var data []byte
	for i := 0; i < 30; i++ {
		time.Sleep(1 * time.Second)
		var err error
		data, err = os.ReadFile(recordingPath)
		if err == nil && len(data) > 0 {
			logger.Info(fmt.Sprintf("RDP recording file read successfully: %s (%d bytes)", recordingPath, len(data)))
			break
		}
		if err != nil {
			logger.Info(fmt.Sprintf("RDP recording file read attempt %d/30 failed: %s", i+1, err.Error()))
		} else {
			logger.Info(fmt.Sprintf("RDP recording file is empty on attempt %d/30", i+1))
		}
	}

	if len(data) > 0 {
		uploaded := false
		for i := 0; i < 10; i++ {
			if err := minio.UploadFile(recordingKey, data); err != nil {
				logger.Error(fmt.Sprintf("RDP recording S3 upload attempt %d failed: %s", i+1, err.Error()))
				time.Sleep(500 * time.Millisecond)
				continue
			}
			logger.Info(fmt.Sprintf("RDP recording uploaded to S3: %s (%d bytes)", recordingKey, len(data)))
			uploaded = true
			break
		}
		if !uploaded {
			logger.Error(fmt.Sprintf("RDP recording S3 upload failed after 10 retries: %s", recordingKey))
		}
		// Clean up local recording directory
		dirPath := fmt.Sprintf("%s/%s.guac", recordBasePath, key)
		if err := os.RemoveAll(dirPath); err != nil {
			logger.Error(fmt.Sprintf("RDP recording cleanup failed: %s", err.Error()))
		} else {
			logger.Info(fmt.Sprintf("RDP recording cleanup success: %s", dirPath))
		}
	} else {
		logger.Error(fmt.Sprintf("RDP recording file not found after 30 retries (30s): %s", recordingPath))
	}

	logger.Info(fmt.Sprintf("RDP session ended for key: %s", key))
}

// rdpSetQuit signals the quit channel (non-blocking).
func rdpSetQuit(ch chan bool) {
	select {
	case ch <- true:
	default:
	}
}
