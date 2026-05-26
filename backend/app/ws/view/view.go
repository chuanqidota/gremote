package view

import (
	"encoding/json"
	"net/http"
	"time"
	"gremote/app/api/params"
	"gremote/app/ws/utils/loginAudit"
	"gremote/app/ws/utils/recordAudit"
	"gremote/pkg/asciinema"
	"gremote/pkg/redis"
	"gremote/pkg/terminal"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type wsHandle struct {
}

var WsHandle = new(wsHandle)

var upgrader = websocket.Upgrader{
	// 解决跨域问题
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	Subprotocols: []string{"guacamole"},
}

func (w wsHandle) Handler(c *gin.Context) {
	// 升级http为ws
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// 没有找到key>无效
	key := c.Param("key")
	if key == "" {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("无效链接"))
		return
	}
	// redis中不存在>属于第二次登录
	if redis.IsConnected(key) {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("链接失效,已经被链接过一次"))
		return
	}

	// 通过key获取redis中>用户信息
	var info params.Info
	err = redis.Get(key, &info)
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("获取登录信息失败"))
		return
	}

	// 自动获取客户端真实IP
	clientIP := c.ClientIP()
	if info.User == "" {
		info.User = clientIP
	}
	if info.Source == "" {
		info.Source = clientIP
	}

	// 登录信息写入到es中
	e := loginAudit.NewEsAudit()
	defer redis.DeleteKey(key)
	auditData := map[string]any{
		"key":       key,
		"startTime": time.Now().Format("2006-01-02 15:04:05"),
		"user":      info.User,
		"source":    info.Source,
		"target":    info.Target,
	}
	e.WriteData(auditData)
	defer e.UpdateEndTime(key)

	// 接受第一次消息
	_, firstMessage, _ := conn.ReadMessage()
	var firstData map[string][]int
	err = json.Unmarshal(firstMessage, &firstData)
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("接收窗口大小失败"))
		return
	}
	resizeData, ok := firstData["resize"]
	if !ok || len(resizeData) < 2 {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("窗口大小数据格式错误"))
		return
	}
	cols := resizeData[0]
	rows := resizeData[1]
	// ssh客户端
	client, err := terminal.Client(info.Username, info.Password, info.Target, info.Port)
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("远程服务器连接失败-用户密码不对"))
		return
	}
	defer client.Close()

	// 初始化终端
	t, err := terminal.NewTerminal(client, cols, rows)
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("远程服务器连接失败-终端初始化失败"))
		return
	}
	defer t.Close()

	// 记录操作到es中
	startTime := time.Now()
	record := recordAudit.NewEsRecord()
	asciinema.WriteHeader(key, cols, rows, startTime, record)

	// 核心交互
	quitChan := make(chan bool, 4)
	esDataChan := make(chan []byte, 1024)
	go t.ReceiveWsMsg(conn, quitChan, key, startTime, record)      // ws > terminal
	go t.WriteWsMsg(conn, quitChan, esDataChan)                    // terminal > ws & chan
	go t.WriteEsData(quitChan, key, startTime, record, esDataChan) // chan > es
	t.SessionWait(quitChan)
}
