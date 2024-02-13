package view

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"net/http"
	"time"
	"webssh-go/app/api/params"
	"webssh-go/app/ws/utils/loginAudit"
	"webssh-go/pkg/redis"
	"webssh-go/pkg/sshClient"
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
	data := map[string]any{
		"key":       key,
		"startTime": time.Now().Format("2006-01-02 15:04:05"),
		"user":      info.User,
		"source":    info.Source,
		"target":    info.Target,
	}
	e.WriteData(data)
	defer e.UpdateEndTime(key)

	// ssh客户端
	client, err := sshClient.Client(info.Username, info.Password, info.Target, info.Port)
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("远程服务器连接失败"))
		return
	}
	// 接受终端大小 {"resize":[1,2]}
	_, firstMessage, _ := conn.ReadMessage()
	var firstData map[string][]int
	err = json.Unmarshal(firstMessage, &firstData)
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("接收窗口大小失败"))
		return
	}
	cols := firstData["resize"][0]
	rows := firstData["resize"][1]
	session, err := sshClient.Session(client, cols, rows)
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("远程服务器建立session失败"))
		return
	}
	fmt.Println(session)

	// 监听 ws 消息
	for {
		// 从 ws 读取数据
		mt, message, err := conn.ReadMessage()
		if err != nil {
			break
		}
		// 发送消息到远程服务器
		res, _ := sshClient.Write(session, string(message))
		//往 ws 写数据
		fmt.Println("res---", string(res))

		err = conn.WriteMessage(mt, res)
		if err != nil {
			break
		}
	}
}
