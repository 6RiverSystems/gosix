package ginmiddleware

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type WSHandler interface {
	Upgrader() *websocket.Upgrader
	Handle(*gin.Context, *websocket.Conn)
}

func UpgradingHandler(checker func(*gin.Context) WSHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		if handler := checker(c); handler == nil {
			// inspecting the request for upgrade decided it should handle without
			// upgrade and we are done
			return
		} else {
			upgrader := handler.Upgrader()
			conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
			if err != nil {
				// upgrader will have already sent an http response
				c.Abort()
				return
			}
			defer conn.Close()

			handler.Handle(c, conn)
		}
	}
}
