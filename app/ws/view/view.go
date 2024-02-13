package view

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"net/http"
	"time"
	"webssh-go/app/api/params"
	"webssh-go/app/ws/utils/loginAudit"
	"webssh-go/pkg/redis"
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

	// 链接处理
	key := c.Param("key")
	if key == "" {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("无效链接"))
		return
	}

	if !redis.Exist(key) {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("链接失效"))
		return
	}

	var info params.Info
	err = redis.Get(key, &info)
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("获取登录信息失败"))
		return
	}
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

	// 监听 ws 消息
	for {
		// 从 ws 读取数据
		mt, message, err := conn.ReadMessage()
		if err != nil {
			break
		}
		//往 ws 写数据
		err = conn.WriteMessage(mt, message)
		if err != nil {
			break
		}
	}

}
