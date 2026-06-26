package distribution

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// DistributorRole 分销商角色
type DistributorRole string

const (
	RoleAgent      DistributorRole = "agent"      // 代理商
	RoleReferrer   DistributorRole = "referrer"    // 推荐人
	RoleReseller   DistributorRole = "reseller"    // 经销商
	RoleAffiliate  DistributorRole = "affiliate"   // 联盟推广
)

// CommissionType 佣金类型
type CommissionType string

const (
	CommissionPercent   CommissionType = "percent"    // 百分比
	CommissionFixed     CommissionType = "fixed"      // 固定金额
	CommissionTiered    CommissionType = "tiered"      // 阶梯式
)

// Distributor 分销商
type Distributor struct {
	ID           string          `json:"id"`
	UserID       string          `json:"user_id"`
	TenantID     string          `json:"tenant_id"`
	Role         DistributorRole `json:"role"`
	ReferralCode string          `json:"referral_code"` // 推广码
	ParentID     string          `json:"parent_id,omitempty"` // 上级分销商ID
	Level        int             `json:"level"`                // 层级（1=一级代理，2=二级代理）
	Status       string          `json:"status"`               // active, suspended, terminated

	// 佣金配置
	CommissionType   CommissionType `json:"commission_type"`
	CommissionRate   float64       `json:"commission_rate"`   // 百分比或固定金额
	CommissionConfig string        `json:"commission_config,omitempty"` // 阶梯配置JSON

	// 统计
	TotalReferred       int     `json:"total_referred"`        // 推荐总人数
	TotalRevenue        int64   `json:"total_revenue"`         // 带来的总营收（分）
	TotalCommission     int64   `json:"total_commission"`      // 累计佣金（分）
	PendingCommission   int64   `json:"pending_commission"`    // 待结算佣金（分）
	WithdrawnCommission int64   `json:"withdrawn_commission"`  // 已提现佣金
	NewReferralsThisMonth int   `json:"new_referrals_month"`   // 本月新增推荐
	TotalClicks         int     `json:"total_clicks"`          // 总点击数
	ConversionRate      float64 `json:"conversion_rate"`       // 转化率

	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
}

// CustomLink 自定义推广链接
type CustomLink struct {
	ID            string `json:"id"`
	DistributorID string `json:"distributor_id"`
	Name          string `json:"name"`
	URL           string `json:"url"`
	Clicks        int    `json:"clicks"`
	Registrations int    `json:"registrations"`
	CreatedAt     int64  `json:"created_at"`
}

// WithdrawRecord 提现记录
type WithdrawRecord struct {
	ID            string  `json:"id"`
	UserID        string  `json:"user_id"`
	Amount        float64 `json:"amount"`
	Method        string  `json:"method"`
	Account       string  `json:"account"`
	RealName      string  `json:"real_name"`
	Status        string  `json:"status"` // pending, processing, completed, failed, cancelled
	Note          string  `json:"note"`
	RejectedReason string `json:"rejected_reason,omitempty"`
	CreatedAt     int64   `json:"created_at"`
	UpdatedAt     int64   `json:"updated_at"`
}

// ReferralUser 推荐用户信息
type ReferralUser struct {
	ID           string  `json:"id"`
	Username     string  `json:"username"`
	Email        string  `json:"email"`
	RegisteredAt int64   `json:"registered_at"`
	LastActive   int64   `json:"last_active"`
	TotalSpent   float64 `json:"total_spent"`
	Contribution float64 `json:"contribution"`
	Status       string  `json:"status"`
}

// PromotionalMaterial 推广素材
type PromotionalMaterial struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Type        string `json:"type"` // image, video, copy
	Category    string `json:"category"`
	URL         string `json:"url"`
	Size        string `json:"size"`
	Downloads   int    `json:"downloads"`
}

// CommissionRecord 佣金记录
type CommissionRecord struct {
	ID             string    `json:"id"`
	DistributorID  string    `json:"distributor_id"`
	ReferralUserID string    `json:"referral_user_id"` // 被推荐用户ID
	OrderNo        string    `json:"order_no"`          // 关联的订单号
	OrderAmount    int64     `json:"order_amount"`      // 订单金额（分）
	CommissionRate float64   `json:"commission_rate"`
	CommissionAmt  int64     `json:"commission_amt"`     // 佣金金额（分）
	Status         string    `json:"status"`             // pending, settled, cancelled
	Period         string    `json:"period"`             // 结算周期 YYYY-MM
	SettledAt      *int64    `json:"settled_at,omitempty"`
	CreatedAt      int64     `json:"created_at"`
}

// DistributionService 分销服务
type DistributionService struct {
	logger *zap.Logger
	rdb    *redis.Client
}

// NewDistributionService 创建分销服务
func NewDistributionService(logger *zap.Logger, rdb *redis.Client) *DistributionService {
	return &DistributionService{logger: logger, rdb: rdb}
}

// RegisterDistributor 注册分销商
func (s *DistributionService) RegisterDistributor(ctx context.Context, dist *Distributor) error {
	dist.ID = uuid.New().String()
	dist.ReferralCode = s.generateReferralCode()
	dist.Status = "active"
	dist.CreatedAt = time.Now().Unix()
	dist.UpdatedAt = dist.CreatedAt

	if dist.Level == 0 {
		dist.Level = 1
	}
	if dist.CommissionType == "" {
		dist.CommissionType = CommissionPercent
	}
	if dist.CommissionRate == 0 {
		dist.CommissionRate = 0.15 // 默认15%
	}

	// 存储
	key := fmt.Sprintf("distributor:%s", dist.ID)
	if err := s.rdb.HSet(ctx, key, map[string]interface{}{
		"id":                dist.ID,
		"user_id":           dist.UserID,
		"tenant_id":         dist.TenantID,
		"role":              string(dist.Role),
		"referral_code":     dist.ReferralCode,
		"parent_id":         dist.ParentID,
		"level":             dist.Level,
		"status":            dist.Status,
		"commission_type":   string(dist.CommissionType),
		"commission_rate":   dist.CommissionRate,
		"total_referred":    0,
		"total_revenue":     0,
		"total_commission":  0,
		"pending_commission": 0,
		"created_at":        dist.CreatedAt,
		"updated_at":        dist.UpdatedAt,
	}).Err(); err != nil {
		return fmt.Errorf("store distributor: %w", err)
	}

	// 索引
	s.rdb.Set(ctx, fmt.Sprintf("referral:%s", dist.ReferralCode), dist.ID, 0)
	s.rdb.SAdd(ctx, fmt.Sprintf("distributor:user:%s", dist.UserID), dist.ID)
	s.rdb.SAdd(ctx, fmt.Sprintf("distributor:tenant:%s", dist.TenantID), dist.ID)

	s.logger.Info("distributor registered",
		zap.String("id", dist.ID),
		zap.String("referral_code", dist.ReferralCode),
		zap.String("role", string(dist.Role)),
	)
	return nil
}

// GetDistributor 获取分销商信息
func (s *DistributionService) GetDistributor(ctx context.Context, id string) (*Distributor, error) {
	key := fmt.Sprintf("distributor:%s", id)
	data, err := s.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("distributor not found: %s", id)
	}

	return &Distributor{
		ID:                data["id"],
		UserID:            data["user_id"],
		TenantID:          data["tenant_id"],
		Role:              DistributorRole(data["role"]),
		ReferralCode:      data["referral_code"],
		ParentID:          data["parent_id"],
		Level:             parseInt(data["level"]),
		Status:            data["status"],
		CommissionType:    CommissionType(data["commission_type"]),
		CommissionRate:    parseFloat(data["commission_rate"]),
		TotalReferred:     parseInt(data["total_referred"]),
		TotalRevenue:      parseInt64(data["total_revenue"]),
		TotalCommission:   parseInt64(data["total_commission"]),
		PendingCommission: parseInt64(data["pending_commission"]),
		CreatedAt:         parseInt64(data["created_at"]),
	}, nil
}

// GetDistributorByReferralCode 通过推广码获取分销商
func (s *DistributionService) GetDistributorByReferralCode(ctx context.Context, code string) (*Distributor, error) {
	distID, err := s.rdb.Get(ctx, fmt.Sprintf("referral:%s", code)).Result()
	if err != nil {
		return nil, fmt.Errorf("invalid referral code: %s", code)
	}
	return s.GetDistributor(ctx, distID)
}

// RecordReferral 记录推荐关系
func (s *DistributionService) RecordReferral(ctx context.Context, referralCode, newUserID string) error {
	dist, err := s.GetDistributorByReferralCode(ctx, referralCode)
	if err != nil {
		return err
	}

	// 存储推荐关系
	refKey := fmt.Sprintf("referral:user:%s", newUserID)
	s.rdb.Set(ctx, refKey, dist.ID, 0)

	// 更新分销商统计
	distKey := fmt.Sprintf("distributor:%s", dist.ID)
	s.rdb.HIncrBy(ctx, distKey, "total_referred", 1)

	s.logger.Info("referral recorded",
		zap.String("distributor_id", dist.ID),
		zap.String("new_user_id", newUserID),
	)
	return nil
}

// CalculateCommission 计算佣金
func (s *DistributionService) CalculateCommission(ctx context.Context, dist *Distributor, orderAmount int64) int64 {
	switch dist.CommissionType {
	case CommissionPercent:
		return int64(float64(orderAmount) * dist.CommissionRate)
	case CommissionFixed:
		return int64(dist.CommissionRate * 100) // CommissionRate as fixed amount in yuan
	default:
		return int64(float64(orderAmount) * dist.CommissionRate)
	}
}

// RecordCommission 记录佣金
func (s *DistributionService) RecordCommission(ctx context.Context, distID, referralUserID, orderNo string, orderAmount int64) error {
	dist, err := s.GetDistributor(ctx, distID)
	if err != nil {
		return err
	}

	commissionAmt := s.CalculateCommission(ctx, dist, orderAmount)
	if commissionAmt <= 0 {
		return nil
	}

	record := &CommissionRecord{
		ID:             uuid.New().String(),
		DistributorID:  distID,
		ReferralUserID: referralUserID,
		OrderNo:        orderNo,
		OrderAmount:    orderAmount,
		CommissionRate: dist.CommissionRate,
		CommissionAmt:  commissionAmt,
		Status:         "pending",
		Period:         time.Now().Format("2006-01"),
		CreatedAt:      time.Now().Unix(),
	}

	// 存储佣金记录
	recKey := fmt.Sprintf("commission:%s", record.ID)
	s.rdb.HSet(ctx, recKey, map[string]interface{}{
		"id":               record.ID,
		"distributor_id":   record.DistributorID,
		"referral_user_id": record.ReferralUserID,
		"order_no":         record.OrderNo,
		"order_amount":     record.OrderAmount,
		"commission_rate":  record.CommissionRate,
		"commission_amt":   record.CommissionAmt,
		"status":           record.Status,
		"period":           record.Period,
		"created_at":       record.CreatedAt,
	})

	// 索引
	s.rdb.ZAdd(ctx, fmt.Sprintf("commission:distributor:%s", distID), &redis.Z{
		Score:  float64(record.CreatedAt),
		Member: record.ID,
	})
	s.rdb.SAdd(ctx, fmt.Sprintf("commission:period:%s", record.Period), record.ID)

	// 更新分销商统计
	distKey := fmt.Sprintf("distributor:%s", distID)
	s.rdb.HIncrBy(ctx, distKey, "total_revenue", orderAmount)
	s.rdb.HIncrBy(ctx, distKey, "total_commission", commissionAmt)
	s.rdb.HIncrBy(ctx, distKey, "pending_commission", commissionAmt)

	// 如果有上级分销商，给上级也分一部分（通常3-5%）
	if dist.ParentID != "" {
		parent, err := s.GetDistributor(ctx, dist.ParentID)
		if err == nil && parent.Status == "active" {
			parentCommission := int64(float64(commissionAmt) * 0.2) // 上级分20%的佣金
			if parentCommission > 0 {
				s.RecordCommission(ctx, parent.ID, referralUserID, orderNo, orderCommissionForParent(orderAmount))
			}
		}
	}

	s.logger.Info("commission recorded",
		zap.String("distributor_id", distID),
		zap.Int64("commission", commissionAmt),
		zap.Int64("order_amount", orderAmount),
	)
	return nil
}

// SettleCommissions 结算佣金
func (s *DistributionService) SettleCommissions(ctx context.Context, period string) (int64, error) {
	periodKey := fmt.Sprintf("commission:period:%s", period)
	recordIDs, err := s.rdb.SMembers(ctx, periodKey).Result()
	if err != nil {
		return 0, err
	}

	var totalSettled int64
	now := time.Now().Unix()

	for _, id := range recordIDs {
		recKey := fmt.Sprintf("commission:%s", id)
		data, err := s.rdb.HGetAll(ctx, recKey).Result()
		if err != nil || len(data) == 0 {
			continue
		}

		if data["status"] != "pending" {
			continue
		}

		// 更新状态
		s.rdb.HSet(ctx, recKey, map[string]interface{}{
			"status":     "settled",
			"settled_at": now,
		})

		// 减少待结算佣金
		distKey := fmt.Sprintf("distributor:%s", data["distributor_id"])
		amt := parseInt64(data["commission_amt"])
		s.rdb.HIncrBy(ctx, distKey, "pending_commission", -amt)

		totalSettled += amt
	}

	s.logger.Info("commissions settled",
		zap.String("period", period),
		zap.Int64("total", totalSettled),
	)
	return totalSettled, nil
}

// ListDistributors 获取租户下所有分销商
func (s *DistributionService) ListDistributors(ctx context.Context, tenantID string) ([]*Distributor, error) {
	setKey := fmt.Sprintf("distributor:tenant:%s", tenantID)
	ids, err := s.rdb.SMembers(ctx, setKey).Result()
	if err != nil {
		return nil, err
	}

	var distributors []*Distributor
	for _, id := range ids {
		dist, err := s.GetDistributor(ctx, id)
		if err != nil {
			continue
		}
		distributors = append(distributors, dist)
	}
	return distributors, nil
}

// ListCommissionRecords 获取佣金记录
func (s *DistributionService) ListCommissionRecords(ctx context.Context, distID string, offset, limit int64) ([]*CommissionRecord, error) {
	zKey := fmt.Sprintf("commission:distributor:%s", distID)
	ids, err := s.rdb.ZRevRange(ctx, zKey, offset, offset+limit-1).Result()
	if err != nil {
		return nil, err
	}

	var records []*CommissionRecord
	for _, id := range ids {
		recKey := fmt.Sprintf("commission:%s", id)
		data, err := s.rdb.HGetAll(ctx, recKey).Result()
		if err != nil || len(data) == 0 {
			continue
		}

		records = append(records, &CommissionRecord{
			ID:             data["id"],
			DistributorID:  data["distributor_id"],
			ReferralUserID: data["referral_user_id"],
			OrderNo:        data["order_no"],
			OrderAmount:    parseInt64(data["order_amount"]),
			CommissionRate: parseFloat(data["commission_rate"]),
			CommissionAmt:  parseInt64(data["commission_amt"]),
			Status:         data["status"],
			Period:         data["period"],
			CreatedAt:      parseInt64(data["created_at"]),
		})
	}
	return records, nil
}

// ==================== 分销商查询 ====================

// GetDistributorByUserID 通过用户ID获取分销商ID列表
func (s *DistributionService) GetDistributorByUserID(ctx context.Context, userID string) ([]string, error) {
	setKey := fmt.Sprintf("distributor:user:%s", userID)
	ids, err := s.rdb.SMembers(ctx, setKey).Result()
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// GetDistributorByUserIDCtx 从上下文获取用户ID对应的分销商
func (s *DistributionService) GetDistributorByUserIDCtx(ctx context.Context, userID string) (*Distributor, error) {
	ids, err := s.GetDistributorByUserID(ctx, userID)
	if err != nil || len(ids) == 0 {
		return nil, fmt.Errorf("distributor not found for user: %s", userID)
	}
	return s.GetDistributor(ctx, ids[0])
}

// ==================== 推荐用户管理 ====================

// ListReferrals 列出推荐用户
func (s *DistributionService) ListReferrals(ctx context.Context, distID string, offset, limit int64) []ReferralUser {
	zKey := fmt.Sprintf("referral:distributor:%s", distID)
	userIDs, err := s.rdb.ZRevRange(ctx, zKey, offset, offset+limit-1).Result()
	if err != nil {
		return nil
	}

	var users []ReferralUser
	for _, uid := range userIDs {
		u := s.getReferralUserInfo(ctx, uid, distID)
		if u != nil {
			users = append(users, *u)
		}
	}
	return users
}

// ListReferralsWithCount 列出推荐用户并返回总数
func (s *DistributionService) ListReferralsWithCount(ctx context.Context, distID string, offset, limit int64) ([]ReferralUser, int64) {
	zKey := fmt.Sprintf("referral:distributor:%s", distID)
	total, err := s.rdb.ZCard(ctx, zKey).Result()
	if err != nil {
		total = 0
	}

	userIDs, err := s.rdb.ZRevRange(ctx, zKey, offset, offset+limit-1).Result()
	if err != nil {
		return nil, total
	}

	var users []ReferralUser
	for _, uid := range userIDs {
		u := s.getReferralUserInfo(ctx, uid, distID)
		if u != nil {
			users = append(users, *u)
		}
	}
	return users, total
}

func (s *DistributionService) getReferralUserInfo(ctx context.Context, userID, distID string) *ReferralUser {
	// 从 referral:user:{userID} 获取基本信息
	userData, err := s.rdb.HGetAll(ctx, fmt.Sprintf("referral:userdata:%s", userID)).Result()
	if err != nil || len(userData) == 0 {
		return &ReferralUser{
			ID:           userID,
			Username:     fmt.Sprintf("user_%s", userID[:8]),
			Email:        fmt.Sprintf("user***@mail.com"),
			RegisteredAt: 0,
			LastActive:   0,
			TotalSpent:   0,
			Contribution: 0,
			Status:       "active",
		}
	}

	totalSpent, _ := strconv.ParseFloat(userData["total_spent"], 64)
	contribution, _ := strconv.ParseFloat(userData["contribution"], 64)
	registeredAt, _ := strconv.ParseInt(userData["registered_at"], 10, 64)
	lastActive, _ := strconv.ParseInt(userData["last_active"], 10, 64)

	return &ReferralUser{
		ID:           userID,
		Username:     userData["username"],
		Email:        userData["email"],
		RegisteredAt: registeredAt,
		LastActive:   lastActive,
		TotalSpent:   totalSpent,
		Contribution: contribution,
		Status:       userData["status"],
	}
}

// ==================== 自定义链接管理 ====================

// ListCustomLinks 列出自定义链接
func (s *DistributionService) ListCustomLinks(ctx context.Context, distID string) []CustomLink {
	setKey := fmt.Sprintf("customlink:distributor:%s", distID)
	linkIDs, err := s.rdb.SMembers(ctx, setKey).Result()
	if err != nil {
		return nil
	}

	var links []CustomLink
	for _, id := range linkIDs {
		data, err := s.rdb.HGetAll(ctx, fmt.Sprintf("customlink:%s", id)).Result()
		if err != nil || len(data) == 0 {
			continue
		}
		links = append(links, CustomLink{
			ID:            data["id"],
			DistributorID: data["distributor_id"],
			Name:          data["name"],
			URL:           data["url"],
			Clicks:        parseInt(data["clicks"]),
			Registrations: parseInt(data["registrations"]),
			CreatedAt:     parseInt64(data["created_at"]),
		})
	}
	return links
}

// CreateCustomLink 创建自定义链接
func (s *DistributionService) CreateCustomLink(ctx context.Context, userID, name, targetPage, note string) (*CustomLink, error) {
	dist, err := s.GetDistributorByUserIDCtx(ctx, userID)
	if err != nil {
		return nil, err
	}

	linkID := uuid.New().String()
	now := time.Now().Unix()
	url := fmt.Sprintf("https://tokenshopai.com/register?ref=%s&src=%s", dist.ReferralCode, strings.ReplaceAll(strings.ToLower(name), " ", "_"))

	linkKey := fmt.Sprintf("customlink:%s", linkID)
	s.rdb.HSet(ctx, linkKey, map[string]interface{}{
		"id":             linkID,
		"distributor_id": dist.ID,
		"name":           name,
		"url":            url,
		"clicks":         0,
		"registrations":  0,
		"created_at":     now,
	})
	s.rdb.SAdd(ctx, fmt.Sprintf("customlink:distributor:%s", dist.ID), linkID)

	return &CustomLink{
		ID:            linkID,
		DistributorID: dist.ID,
		Name:          name,
		URL:           url,
		Clicks:        0,
		Registrations: 0,
		CreatedAt:     now,
	}, nil
}

// DeleteCustomLink 删除自定义链接
func (s *DistributionService) DeleteCustomLink(ctx context.Context, userID, linkID string) error {
	dist, err := s.GetDistributorByUserIDCtx(ctx, userID)
	if err != nil {
		return err
	}

	// 验证链接属于该分销商
	data, err := s.rdb.HGetAll(ctx, fmt.Sprintf("customlink:%s", linkID)).Result()
	if err != nil || data["distributor_id"] != dist.ID {
		return fmt.Errorf("link not found or unauthorized")
	}

	s.rdb.Del(ctx, fmt.Sprintf("customlink:%s", linkID))
	s.rdb.SRem(ctx, fmt.Sprintf("customlink:distributor:%s", dist.ID), linkID)
	return nil
}

// ==================== 提现管理 ====================

// ListWithdrawRecords 列出提现记录
func (s *DistributionService) ListWithdrawRecords(ctx context.Context, userID string, offset, limit int64) []WithdrawRecord {
	zKey := fmt.Sprintf("withdraw:user:%s", userID)
	recordIDs, err := s.rdb.ZRevRange(ctx, zKey, offset, offset+limit-1).Result()
	if err != nil {
		return nil
	}

	var records []WithdrawRecord
	for _, id := range recordIDs {
		data, err := s.rdb.HGetAll(ctx, fmt.Sprintf("withdraw:%s", id)).Result()
		if err != nil || len(data) == 0 {
			continue
		}
		amount, _ := strconv.ParseFloat(data["amount"], 64)
		records = append(records, WithdrawRecord{
			ID:        data["id"],
			UserID:    data["user_id"],
			Amount:    amount,
			Method:    data["method"],
			Account:   data["account"],
			RealName:  data["real_name"],
			Status:    data["status"],
			Note:      data["note"],
			CreatedAt: parseInt64(data["created_at"]),
			UpdatedAt: parseInt64(data["updated_at"]),
		})
	}
	return records
}

// RequestWithdraw 申请提现
func (s *DistributionService) RequestWithdraw(ctx context.Context, userID string, amount float64, method, account, realName, note string) error {
	if amount < 10 {
		return fmt.Errorf("minimum withdraw amount is 10")
	}

	recordID := uuid.New().String()
	now := time.Now().Unix()

	recordKey := fmt.Sprintf("withdraw:%s", recordID)
	s.rdb.HSet(ctx, recordKey, map[string]interface{}{
		"id":         recordID,
		"user_id":    userID,
		"amount":     amount,
		"method":     method,
		"account":    account,
		"real_name":  realName,
		"status":     "pending",
		"note":       note,
		"created_at": now,
		"updated_at": now,
	})
	s.rdb.ZAdd(ctx, fmt.Sprintf("withdraw:user:%s", userID), &redis.Z{
		Score:  float64(now),
		Member: recordID,
	})
	s.rdb.SAdd(ctx, "withdraw:pending", recordID)

	s.logger.Info("withdraw request created",
		zap.String("user_id", userID),
		zap.Float64("amount", amount),
		zap.String("method", method),
	)
	return nil
}

// CancelWithdraw 取消提现
func (s *DistributionService) CancelWithdraw(ctx context.Context, userID, recordID string) error {
	recordKey := fmt.Sprintf("withdraw:%s", recordID)
	data, err := s.rdb.HGetAll(ctx, recordKey).Result()
	if err != nil || len(data) == 0 {
		return fmt.Errorf("record not found")
	}
	if data["user_id"] != userID {
		return fmt.Errorf("unauthorized")
	}
	if data["status"] != "pending" {
		return fmt.Errorf("only pending withdrawals can be cancelled")
	}

	now := time.Now().Unix()
	s.rdb.HSet(ctx, recordKey, map[string]interface{}{
		"status":     "cancelled",
		"updated_at": now,
	})
	s.rdb.SRem(ctx, "withdraw:pending", recordID)

	return nil
}

// ==================== 推广素材 ====================

// ListPromotionalMaterials 列出推广素材
func (s *DistributionService) ListPromotionalMaterials(ctx context.Context, category string) []PromotionalMaterial {
	var allMaterials = []PromotionalMaterial{
		{ID: "1", Title: "产品介绍横幅", Description: "适合网站顶部展示", Type: "image", Category: "banner", URL: "https://via.placeholder.com/1920x600/6366f1/ffffff?text=Product+Banner", Size: "1920x600", Downloads: 256},
		{ID: "2", Title: "促销活动海报", Description: "适合社交媒体分享", Type: "image", Category: "poster", URL: "https://via.placeholder.com/1080x1920/ec4899/ffffff?text=Promotion+Poster", Size: "1080x1920", Downloads: 189},
		{ID: "3", Title: "产品演示视频", Description: "展示产品核心功能", Type: "video", Category: "video", URL: "", Size: "MP4 50MB", Downloads: 123},
		{ID: "4", Title: "用户案例海报", Description: "真实用户使用案例", Type: "image", Category: "poster", URL: "https://via.placeholder.com/1080x1920/8b5cf6/ffffff?text=Case+Study", Size: "1080x1920", Downloads: 145},
		{ID: "5", Title: "功能介绍视频", Description: "详细功能使用教程", Type: "video", Category: "video", URL: "", Size: "MP4 80MB", Downloads: 98},
		{ID: "6", Title: "品牌宣传横幅", Description: "品牌形象展示", Type: "image", Category: "banner", URL: "https://via.placeholder.com/1920x600/10b981/ffffff?text=Brand+Banner", Size: "1920x600", Downloads: 167},
	}

	if category == "all" || category == "" {
		return allMaterials
	}

	var filtered []PromotionalMaterial
	for _, m := range allMaterials {
		if m.Category == category {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

// ==================== 分销商资料更新 ====================

// UpdateDistributorProfile 更新分销商资料
func (s *DistributionService) UpdateDistributorProfile(ctx context.Context, userID string, commissionRate float64) error {
	dist, err := s.GetDistributorByUserIDCtx(ctx, userID)
	if err != nil {
		return err
	}

	if commissionRate > 0 {
		if commissionRate > 1.0 {
			commissionRate = commissionRate / 100
		}
		distKey := fmt.Sprintf("distributor:%s", dist.ID)
		s.rdb.HSet(ctx, distKey, "commission_rate", commissionRate)
	}

	return nil
}

// ==================== 辅助函数 ====================

func (s *DistributionService) generateReferralCode() string {
	return fmt.Sprintf("TH%s", uuid.New().String()[:8])
}

func parseInt(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

func parseInt64(s string) int64 {
	var n int64
	fmt.Sscanf(s, "%d", &n)
	return n
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

func orderCommissionForParent(amount int64) int64 {
	return int64(float64(amount) * 0.03) // 3% of order
}
