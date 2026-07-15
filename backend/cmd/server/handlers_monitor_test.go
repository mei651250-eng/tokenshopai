package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/tokenhub/backend/internal/monitor"
	"go.uber.org/zap"
)

// TestHandleMonitorWebSocket_NoUpgrade 对普通（非 WebSocket 升级）请求，
// handler 应安全返回而不 panic（升级失败被打断）。这是对监控 handler 的冒烟测试。
func TestHandleMonitorWebSocket_NoUpgrade(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/ws/monitor", nil)
	svc := monitor.NewMonitorService(zap.NewNop())

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("handler 不应 panic: %v", r)
		}
	}()
	handleMonitorWebSocket(c, svc, zap.NewNop())
}
