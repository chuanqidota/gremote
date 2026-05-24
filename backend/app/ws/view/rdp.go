package view

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// RDPHandler handles RDP WebSocket connections (stub - full implementation in Task 4)
func (w wsHandle) RDPHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	_ = conn.WriteMessage(websocket.TextMessage, []byte("RDP handler not implemented"))
}
