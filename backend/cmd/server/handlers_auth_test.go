package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// TestHandleLogin_InvalidJSON 校验在请求体非法 JSON 时尽早返回 400，
// 且不会触达 jwtManager / db（此处以 nil 传入，证明输入校验在依赖使用之前）。
func TestHandleLogin_InvalidJSON(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("not-json"))
	handleLogin(c, nil, nil, zap.NewNop())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("非法 JSON 应返回 400, 实际 %d", w.Code)
	}
}
