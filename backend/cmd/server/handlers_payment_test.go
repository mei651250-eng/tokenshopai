package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/tokenhub/backend/internal/payment"
	"go.uber.org/zap"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	m.Run()
}

// fakePaymentSvc 是 paymentCallbackHandler 的测试假实现
type fakePaymentSvc struct {
	err        error
	gotChannel payment.PaymentChannel
	gotData    []byte
	gotSign    string
}

func (f *fakePaymentSvc) HandleCallback(ctx context.Context, channel payment.PaymentChannel, data []byte, sign string) error {
	f.gotChannel = channel
	f.gotData = data
	f.gotSign = sign
	return f.err
}

func doPaymentRequest(t *testing.T, channel payment.PaymentChannel, body, signHeader, signValue string) (*httptest.ResponseRecorder, *fakePaymentSvc) {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/callback", strings.NewReader(body))
	if signHeader != "" {
		req.Header.Set(signHeader, signValue)
	}
	c.Request = req

	svc := &fakePaymentSvc{}
	handlePaymentCallback(c, svc, channel, zap.NewNop())
	return w, svc
}

func TestHandlePaymentCallback_AlipaySuccess(t *testing.T) {
	w, svc := doPaymentRequest(t, payment.ChannelAlipay, "payload", "X-Signature", "sig-abc")
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200, 实际 %d", w.Code)
	}
	if w.Body.String() != "success" {
		t.Fatalf("支付宝应返回 success, 实际 %q", w.Body.String())
	}
	if svc.gotChannel != payment.ChannelAlipay || svc.gotSign != "sig-abc" || string(svc.gotData) != "payload" {
		t.Fatalf("回调参数未正确传递: %+v", svc)
	}
}

func TestHandlePaymentCallback_WeChatSignAndResponse(t *testing.T) {
	w, svc := doPaymentRequest(t, payment.ChannelWeChatPay, "wx-payload", "Wechatpay-Signature", "wx-sig")
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200, 实际 %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "SUCCESS") {
		t.Fatalf("微信应返回 SUCCESS, 实际 %q", w.Body.String())
	}
	if svc.gotSign != "wx-sig" {
		t.Fatalf("应从 Wechatpay-Signature 头取签名, 实际 %q", svc.gotSign)
	}
}

func TestHandlePaymentCallback_DefaultChannelResponse(t *testing.T) {
	w, _ := doPaymentRequest(t, payment.ChannelStripe, "stripe-payload", "X-Signature", "s")
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200, 实际 %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "ok") {
		t.Fatalf("默认渠道应返回 status ok, 实际 %q", w.Body.String())
	}
}

func TestHandlePaymentCallback_VerifyError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/callback", strings.NewReader("p"))
	svc := &fakePaymentSvc{err: context.Canceled}
	handlePaymentCallback(c, svc, payment.ChannelAlipay, zap.NewNop())
	if w.Code != http.StatusBadRequest {
		t.Fatalf("校验失败应返回 400, 实际 %d", w.Code)
	}
}
