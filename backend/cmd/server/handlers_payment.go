package main

import (
	"context"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tokenhub/backend/internal/payment"
	"go.uber.org/zap"
)

// paymentCallbackHandler 是支付回调处理所需的最小接口，便于在测试中注入假实现。
type paymentCallbackHandler interface {
	HandleCallback(ctx context.Context, channel payment.PaymentChannel, data []byte, sign string) error
}

// handlePaymentCallback 处理支付回调
func handlePaymentCallback(c *gin.Context, svc paymentCallbackHandler, channel payment.PaymentChannel, logger *zap.Logger) {
	data, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "read body failed"})
		return
	}

	sign := c.GetHeader("X-Signature")
	if sign == "" {
		sign = c.GetHeader("Wechatpay-Signature")
	}
	if sign == "" {
		sign = c.Query("sign")
	}

	if err := svc.HandleCallback(c.Request.Context(), channel, data, sign); err != nil {
		logger.Error("payment callback failed", zap.String("channel", string(channel)), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 各渠道回调响应格式不同
	switch channel {
	case payment.ChannelAlipay:
		c.String(http.StatusOK, "success")
	case payment.ChannelWeChatPay:
		c.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "message": "OK"})
	default:
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}
