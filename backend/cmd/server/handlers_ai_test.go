package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// TestHandleChatCompletion_InvalidJSON 校验非法 JSON 时尽早返回 400，
// 各 service 依赖以 nil 传入，证明输入校验（ShouldBindJSON）在依赖使用之前。
func TestHandleChatCompletion_InvalidJSON(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader("broken"))
	handleChatCompletionWithChannel(c, nil, nil, nil, nil, nil, zap.NewNop())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("非法 JSON 应返回 400, 实际 %d", w.Code)
	}
}
