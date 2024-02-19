package view

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"net/http"
	"time"
	"webssh-go/app/api/params"
	"webssh-go/app/ws/utils/loginAudit"
	"webssh-go/app/ws/utils/recordAudit"
	"webssh-go/pkg/asciinema"
	"webssh-go/pkg/logger"
	"webssh-go/pkg/redis"
	"webssh-go/pkg/terminal"
)

type wsHandle struct {
}

var WsHandle = new(wsHandle)

var upgrader = websocket.Upgrader{
	// 解决跨域问题
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
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
	if !redis.Exist(key) {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("链接失效"))
		return
	}
	// 通过key获取redis中>用户信息
	var info params.Info
	err = redis.Get(key, &info)
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("获取登录信息失败"))
		return
	}
	// 获取成功以后删除redis中的key
	_ = redis.DeleteKey(key)

	// 登录信息写入到es中
	e := loginAudit.NewEsAudit()
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
	cols := firstData["resize"][0]
	rows := firstData["resize"][1]

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
	logger.Info("websocket连接成功-等待终端发送消息")

	// 记录操作到es中
	startTime := time.Now()
	record := recordAudit.NewEsRecord()
	asciinema.WriteHeader(key, cols, rows, startTime, record)

	// 核心交互
	quitChan := make(chan bool, 4)
	esDataChan := make(chan []byte)
	go t.ReceiveWsMsg(conn, quitChan, key, startTime, record) // ws > terminal
	go t.WriteWsMsg(conn, quitChan, esDataChan)               // terminal > ws
	go t.WriteEsData(quitChan, key, startTime, record, esDataChan)
	go t.SessionWait(quitChan) // 关闭session
	<-quitChan

}
