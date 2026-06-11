package channel

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// HealthChecker 渠道健康检查器
type HealthChecker struct {
	service  *ChannelService
	logger   *zap.Logger
	interval time.Duration
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(service *ChannelService, logger *zap.Logger, interval time.Duration) *HealthChecker {
	if interval == 0 {
		interval = 5 * time.Minute
	}
	return &HealthChecker{
		service:  service,
		logger:   logger,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start 启动健康检查
func (h *HealthChecker) Start() {
	h.wg.Add(1)
	go h.run()
	h.logger.Info("channel health checker started", zap.Duration("interval", h.interval))
}

// Stop 停止健康检查
func (h *HealthChecker) Stop() {
	close(h.stopCh)
	h.wg.Wait()
	h.logger.Info("channel health checker stopped")
}

func (h *HealthChecker) run() {
	defer h.wg.Done()

	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	// 启动后先检查一次
	h.checkAll()

	for {
		select {
		case <-ticker.C:
			h.checkAll()
		case <-h.stopCh:
			return
		}
	}
}

func (h *HealthChecker) checkAll() {
	ctx := context.Background()
	results := h.service.BatchTestChannels(ctx)

	successCount := 0
	failCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		} else {
			failCount++
			h.logger.Warn("channel health check failed",
				zap.String("channel_id", r.ChannelID),
				zap.String("error", r.Error),
			)
		}
	}

	h.logger.Info("channel health check completed",
		zap.Int("total", len(results)),
		zap.Int("success", successCount),
		zap.Int("failed", failCount),
	)
}
