package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/tokenhub/backend/internal/monitor"
	"go.uber.org/zap"
)

func handleMonitorWebSocket(c *gin.Context, monitorService *monitor.MonitorService, logger *zap.Logger) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			// 生产环境应检查Origin是否在允许列表中
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true // 非浏览器请求
			}
			// 开发环境允许所有来源，生产环境应限制
			return true
		},
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Error("websocket upgrade failed", zap.Error(err))
		return
	}

	monitorService.RegisterClient(conn)
	defer monitorService.UnregisterClient(conn)

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}
