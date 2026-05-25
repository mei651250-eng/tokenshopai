package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// ===================== Models =====================

type User struct {
	ID          string    `gorm:"primaryKey;size:36" json:"id"`
	TenantID    string    `gorm:"size:36;index" json:"tenant_id"`
	Email       string    `gorm:"uniqueIndex;size:255" json:"email"`
	DisplayName string    `gorm:"size:100" json:"display_name"`
	Password    string    `gorm:"size:255" json:"-"`
	Role        string    `gorm:"size:50;index" json:"role"` // super_admin, tenant_admin, editor, viewer
	Status      string    `gorm:"size:20;default:active" json:"status"` // active, inactive, banned
	AvatarURL   string    `gorm:"size:500" json:"avatar_url"`
	Phone       string    `gorm:"size:20" json:"phone"`
	Company     string    `gorm:"size:200" json:"company"`
	Bio         string    `gorm:"size:500" json:"bio"`
	TwoFA       bool      `gorm:"default:false" json:"two_fa"`
	LastLogin   time.Time `json:"last_login"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Role struct {
	ID          string   `gorm:"primaryKey;size:36" json:"id"`
	TenantID    string   `gorm:"size:36;index" json:"tenant_id"`
	Name        string   `gorm:"size:50" json:"name"`
	Description string   `gorm:"size:500" json:"description"`
	Permissions string   `gorm:"type:text" json:"permissions"` // JSON array of permission strings
	IsBuiltin   bool     `gorm:"default:false" json:"is_builtin"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type AuditLog struct {
	ID         string    `gorm:"primaryKey;size:36" json:"id"`
	UserID     string    `gorm:"size:36;index" json:"user_id"`
	UserEmail  string    `gorm:"size:255" json:"user_email"`
	Action     string    `gorm:"size:50;index" json:"action"` // create, update, delete, login, logout, export
	Resource   string    `gorm:"size:50;index" json:"resource"` // model, user, tenant, payment, security, setting
	ResourceID string    `gorm:"size:100" json:"resource_id"`
	Detail     string    `gorm:"type:text" json:"detail"`
	IP         string    `gorm:"size:50" json:"ip"`
	UserAgent  string    `gorm:"size:500" json:"user_agent"`
	Level      string    `gorm:"size:20;default:info" json:"level"` // info, warning, error
	CreatedAt  time.Time `gorm:"index" json:"created_at"`
}

type Notification struct {
	ID        string    `gorm:"primaryKey;size:36" json:"id"`
	UserID    string    `gorm:"size:36;index" json:"user_id"`
	Type      string    `gorm:"size:30;index" json:"type"` // system, alert, billing, security, model
	Level     string    `gorm:"size:20" json:"level"` // info, warning, error, success
	Title     string    `gorm:"size:200" json:"title"`
	Message   string    `gorm:"type:text" json:"message"`
	Read      bool      `gorm:"default:false;index" json:"read"`
	Link      string    `gorm:"size:500" json:"link"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

type LoginDevice struct {
	ID         string    `gorm:"primaryKey;size:36" json:"id"`
	UserID     string    `gorm:"size:36;index" json:"user_id"`
	DeviceName string    `gorm:"size:200" json:"device_name"`
	Location   string    `gorm:"size:100" json:"location"`
	IP         string    `gorm:"size:50" json:"ip"`
	UserAgent  string    `gorm:"size:500" json:"user_agent"`
	LastActive time.Time `json:"last_active"`
	IsCurrent  bool      `gorm:"default:false" json:"is_current"`
	CreatedAt  time.Time `json:"created_at"`
}

// ===================== Service =====================

type PlatformService struct {
	db       *gorm.DB
	rdb      *redis.Client
	auditCh  chan *AuditLog // 审计日志异步写入通道
	quitCh   chan struct{}
	wg       sync.WaitGroup
}

func NewPlatformService(db *gorm.DB, rdb *redis.Client) *PlatformService {
	svc := &PlatformService{
		db:      db,
		rdb:     rdb,
		auditCh: make(chan *AuditLog, 1000), // 缓冲1000条
		quitCh:  make(chan struct{}),
	}
	// 启动异步审计日志写入
	svc.wg.Add(1)
	go svc.auditLogWorker()
	return svc
}

// Stop 优雅停止审计日志写入
func (s *PlatformService) Stop() {
	close(s.quitCh)
	s.wg.Wait()
}

// auditLogWorker 异步审计日志写入协程
func (s *PlatformService) auditLogWorker() {
	defer s.wg.Done()
	batch := make([]*AuditLog, 0, 100)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		if err := s.db.CreateInBatches(batch, 100).Error; err != nil {
			// 写入失败时尝试逐条写入
			for _, log := range batch {
				s.db.Create(log)
			}
		}
		batch = batch[:0]
	}

	for {
		select {
		case log := <-s.auditCh:
			batch = append(batch, log)
			if len(batch) >= 100 {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-s.quitCh:
			// 排空剩余日志
			for len(s.auditCh) > 0 {
				log := <-s.auditCh
				batch = append(batch, log)
			}
			flush()
			return
		}
	}
}

func (s *PlatformService) AutoMigrate() error {
	return s.db.AutoMigrate(&User{}, &Role{}, &AuditLog{}, &Notification{}, &LoginDevice{})
}

// ===================== User CRUD =====================

func (s *PlatformService) ListUsers(ctx context.Context, tenantID, search, role, status string, offset, limit int) ([]User, int64, error) {
	var users []User
	var total int64
	q := s.db.WithContext(ctx).Model(&User{})
	if tenantID != "" {
		q = q.Where("tenant_id = ?", tenantID)
	}
	if search != "" {
		q = q.Where("email LIKE ? OR display_name LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if role != "" {
		q = q.Where("role = ?", role)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Offset(offset).Limit(limit).Order("created_at DESC").Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

func (s *PlatformService) CreateUser(ctx context.Context, user *User) error {
	return s.db.WithContext(ctx).Create(user).Error
}

func (s *PlatformService) UpdateUser(ctx context.Context, id string, updates map[string]interface{}) error {
	return s.db.WithContext(ctx).Model(&User{}).Where("id = ?", id).Updates(updates).Error
}

func (s *PlatformService) DeleteUser(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Delete(&User{}, "id = ?", id).Error
}

// ===================== Roles =====================

func (s *PlatformService) ListRoles(ctx context.Context, tenantID string) ([]Role, error) {
	var roles []Role
	q := s.db.WithContext(ctx)
	if tenantID != "" {
		q = q.Where("tenant_id = ? OR is_builtin = true", tenantID)
	}
	if err := q.Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

func (s *PlatformService) CreateRole(ctx context.Context, role *Role) error {
	// 确保Permissions是有效的JSON
	if role.Permissions != "" && role.Permissions[0] != '[' {
		// 不是JSON数组，尝试转换
		var perms []string
		if err := json.Unmarshal([]byte(role.Permissions), &perms); err != nil {
			// 如果不是JSON，包装成JSON数组
			perms = []string{role.Permissions}
			jsonBytes, _ := json.Marshal(perms)
			role.Permissions = string(jsonBytes)
		}
	}
	return s.db.WithContext(ctx).Create(role).Error
}

func (s *PlatformService) UpdateRole(ctx context.Context, id string, updates map[string]interface{}) error {
	// 如果更新了permissions字段，确保是JSON格式
	if perms, ok := updates["permissions"]; ok {
		switch v := perms.(type) {
		case []string:
			jsonBytes, _ := json.Marshal(v)
			updates["permissions"] = string(jsonBytes)
		case []interface{}:
			jsonBytes, _ := json.Marshal(v)
			updates["permissions"] = string(jsonBytes)
		case string:
			if v[0] != '[' {
				arr := []string{v}
				jsonBytes, _ := json.Marshal(arr)
				updates["permissions"] = string(jsonBytes)
			}
		}
	}
	return s.db.WithContext(ctx).Model(&Role{}).Where("id = ? AND is_builtin = false", id).Updates(updates).Error
}

func (s *PlatformService) DeleteRole(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Where("id = ? AND is_builtin = false", id).Delete(&Role{}).Error
}

// ===================== Audit Logs =====================

func (s *PlatformService) CreateAuditLog(ctx context.Context, log *AuditLog) error {
	return s.db.WithContext(ctx).Create(log).Error
}

// RecordAuditLog 异步记录审计日志（推荐使用此方法，非阻塞）
func (s *PlatformService) RecordAuditLog(log *AuditLog) {
	if log.ID == "" {
		log.ID = fmt.Sprintf("audit_%d", time.Now().UnixNano())
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}
	select {
	case s.auditCh <- log:
	default:
		// 通道满了，同步写入
		s.db.Create(log)
	}
}

func (s *PlatformService) ListAuditLogs(ctx context.Context, userEmail, action, resource, level string, startTime, endTime time.Time, offset, limit int) ([]AuditLog, int64, error) {
	var logs []AuditLog
	var total int64
	q := s.db.WithContext(ctx).Model(&AuditLog{})
	if userEmail != "" {
		q = q.Where("user_email LIKE ?", "%"+userEmail+"%")
	}
	if action != "" {
		q = q.Where("action = ?", action)
	}
	if resource != "" {
		q = q.Where("resource = ?", resource)
	}
	if level != "" {
		q = q.Where("level = ?", level)
	}
	if !startTime.IsZero() {
		q = q.Where("created_at >= ?", startTime)
	}
	if !endTime.IsZero() {
		q = q.Where("created_at <= ?", endTime)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Offset(offset).Limit(limit).Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

// ===================== Notifications =====================

func (s *PlatformService) ListNotifications(ctx context.Context, userID, notifType string, read *bool, offset, limit int) ([]Notification, int64, error) {
	var notifs []Notification
	var total int64
	q := s.db.WithContext(ctx).Model(&Notification{}).Where("user_id = ?", userID)
	if notifType != "" {
		q = q.Where("type = ?", notifType)
	}
	if read != nil {
		q = q.Where("read = ?", *read)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Offset(offset).Limit(limit).Order("created_at DESC").Find(&notifs).Error; err != nil {
		return nil, 0, err
	}
	return notifs, total, nil
}

func (s *PlatformService) CreateNotification(ctx context.Context, n *Notification) error {
	return s.db.WithContext(ctx).Create(n).Error
}

func (s *PlatformService) MarkNotificationRead(ctx context.Context, id, userID string) error {
	return s.db.WithContext(ctx).Model(&Notification{}).Where("id = ? AND user_id = ?", id, userID).Update("read", true).Error
}

func (s *PlatformService) MarkAllNotificationsRead(ctx context.Context, userID string) error {
	return s.db.WithContext(ctx).Model(&Notification{}).Where("user_id = ? AND read = false", userID).Update("read", true).Error
}

func (s *PlatformService) DeleteNotification(ctx context.Context, id, userID string) error {
	return s.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&Notification{}).Error
}

func (s *PlatformService) ClearAllNotifications(ctx context.Context, userID string) error {
	return s.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&Notification{}).Error
}

func (s *PlatformService) GetUnreadCount(ctx context.Context, userID string) (int64, error) {
	var count int64
	if err := s.db.WithContext(ctx).Model(&Notification{}).Where("user_id = ? AND read = false", userID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// ===================== Profile =====================

func (s *PlatformService) GetProfile(ctx context.Context, userID string) (*User, error) {
	var user User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		return nil, fmt.Errorf("user not found")
	}
	return &user, nil
}

func (s *PlatformService) UpdateProfile(ctx context.Context, userID string, updates map[string]interface{}) error {
	return s.db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).Updates(updates).Error
}

func (s *PlatformService) UpdateAvatar(ctx context.Context, userID, avatarURL string) error {
	return s.db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).Update("avatar_url", avatarURL).Error
}

func (s *PlatformService) Toggle2FA(ctx context.Context, userID string, enabled bool) error {
	return s.db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).Update("two_fa", enabled).Error
}

// ===================== Login Devices =====================

func (s *PlatformService) ListLoginDevices(ctx context.Context, userID string) ([]LoginDevice, error) {
	var devices []LoginDevice
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Order("last_active DESC").Find(&devices).Error; err != nil {
		return nil, err
	}
	return devices, nil
}

func (s *PlatformService) RevokeLoginDevice(ctx context.Context, id, userID string) error {
	return s.db.WithContext(ctx).Where("id = ? AND user_id = ? AND is_current = false", id, userID).Delete(&LoginDevice{}).Error
}

// ===================== Seed Data =====================

func (s *PlatformService) SeedData(ctx context.Context) {
	// Seed default roles
	roles := []Role{
		{ID: "role-super-admin", Name: "super_admin", Description: "Full system access", Permissions: "[\"*\"]", IsBuiltin: true},
		{ID: "role-tenant-admin", Name: "tenant_admin", Description: "Manage tenant resources", Permissions: "[\"models:read\",\"models:write\",\"billing:read\",\"users:read\",\"settings:read\"]", IsBuiltin: true},
		{ID: "role-editor", Name: "editor", Description: "Edit models and view billing", Permissions: "[\"models:read\",\"models:write\",\"billing:read\"]", IsBuiltin: true},
		{ID: "role-viewer", Name: "viewer", Description: "Read-only access", Permissions: "[\"models:read\",\"billing:read\"]", IsBuiltin: true},
	}
	for _, r := range roles {
		s.db.WithContext(ctx).FirstOrCreate(&r, Role{ID: r.ID})
	}
}
