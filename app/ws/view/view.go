package view

import (
	"net/http"

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
}

func (w wsHandle) Handler(c *gin.Context) {

	// 升级http为ws
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer ws.Close()

	// 监听 ws 消息
	for {
		// 从 ws 读取数据
		mt, message, err := ws.ReadMessage()
		if err != nil {
			break
		}
		//往 ws 写数据
		err = ws.WriteMessage(mt, message)
		if err != nil {
			break
		}
	}

}
