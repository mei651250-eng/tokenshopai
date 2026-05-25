package auth

import (
	"time"
)

// RBAC 角色与权限

// Role 角色
type Role string

const (
	RoleSuperAdmin   Role = "super_admin"   // 超级管理员（平台级）
	RoleTenantAdmin  Role = "tenant_admin"  // 租户管理员
	RoleDeptAdmin    Role = "dept_admin"    // 部门管理员
	RoleDeveloper    Role = "developer"      // 开发者
	RoleViewer       Role = "viewer"         // 只读观察者
	RoleAPIConsumer  Role = "api_consumer"  // API消费者（仅调用API）
)

// Permission 权限
type Permission string

const (
	// 模型管理
	PermModelList    Permission = "model:list"
	PermModelCreate  Permission = "model:create"
	PermModelUpdate  Permission = "model:update"
	PermModelDelete  Permission = "model:delete"
	PermModelRoute   Permission = "model:route"

	// API Key 管理
	PermAPIKeyList   Permission = "apikey:list"
	PermAPIKeyCreate Permission = "apikey:create"
	PermAPIKeyRevoke Permission = "apikey:revoke"

	// 计费
	PermBillingView   Permission = "billing:view"
	PermBillingTopUp  Permission = "billing:topup"
	PermBillingExport Permission = "billing:export"

	// 租户管理
	PermTenantCreate Permission = "tenant:create"
	PermTenantUpdate Permission = "tenant:update"
	PermTenantDelete Permission = "tenant:delete"

	// 用户管理
	PermUserCreate Permission = "user:create"
	PermUserUpdate Permission = "user:update"
	PermUserDelete Permission = "user:delete"

	// 监控
	PermMonitorView  Permission = "monitor:view"
	PermMonitorExport Permission = "monitor:export"

	// 安全
	PermSecurityView Permission = "security:view"
	PermSecurityConfig Permission = "security:config"

	// 报表
	PermReportView   Permission = "report:view"
	PermReportExport Permission = "report:export"

	// 配额管理
	PermQuotaView   Permission = "quota:view"
	PermQuotaConfig Permission = "quota:config"

	// 退款管理
	PermRefundCreate Permission = "refund:create"
	PermRefundReview Permission = "refund:review"

	// 对账
	PermReconView Permission = "reconciliation:view"

	// 分销
	PermDistView   Permission = "distribution:view"
	PermDistManage Permission = "distribution:manage"
)

// RolePermissions 角色权限映射
var RolePermissions = map[Role][]Permission{
	RoleSuperAdmin: {
		PermModelList, PermModelCreate, PermModelUpdate, PermModelDelete, PermModelRoute,
		PermAPIKeyList, PermAPIKeyCreate, PermAPIKeyRevoke,
		PermBillingView, PermBillingTopUp, PermBillingExport,
		PermTenantCreate, PermTenantUpdate, PermTenantDelete,
		PermUserCreate, PermUserUpdate, PermUserDelete,
		PermMonitorView, PermMonitorExport,
		PermSecurityView, PermSecurityConfig,
		PermReportView, PermReportExport,
		PermQuotaView, PermQuotaConfig,
		PermRefundCreate, PermRefundReview,
		PermReconView,
		PermDistView, PermDistManage,
	},
	RoleTenantAdmin: {
		PermModelList, PermModelCreate, PermModelUpdate, PermModelRoute,
		PermAPIKeyList, PermAPIKeyCreate, PermAPIKeyRevoke,
		PermBillingView, PermBillingTopUp, PermBillingExport,
		PermUserCreate, PermUserUpdate, PermUserDelete,
		PermMonitorView, PermMonitorExport,
		PermSecurityView,
		PermReportView, PermReportExport,
		PermQuotaView,
		PermRefundCreate, PermRefundReview,
		PermReconView,
		PermDistView, PermDistManage,
	},
	RoleDeptAdmin: {
		PermModelList, PermModelRoute,
		PermAPIKeyList, PermAPIKeyCreate,
		PermBillingView,
		PermUserCreate, PermUserUpdate,
		PermMonitorView,
		PermReportView,
	},
	RoleDeveloper: {
		PermModelList, PermModelRoute,
		PermAPIKeyList, PermAPIKeyCreate,
		PermBillingView,
		PermMonitorView,
	},
	RoleViewer: {
		PermModelList,
		PermBillingView,
		PermMonitorView,
		PermReportView,
	},
	RoleAPIConsumer: {
		PermModelRoute,
	},
}

// User 用户
type User struct {
	ID           string    `json:"id" gorm:"primaryKey"`
	TenantID     string    `json:"tenant_id" gorm:"index"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone,omitempty"`
	PasswordHash string    `json:"-"`
	Role         Role      `json:"role"`
	Status       UserStatus `json:"status"`
	DepartmentID string    `json:"department_id,omitempty"`
	Language     string    `json:"language"`
	Timezone     string    `json:"timezone"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type UserStatus string

const (
	UserActive   UserStatus = "active"
	UserDisabled UserStatus = "disabled"
	UserLocked   UserStatus = "locked"
)

// APIKey API密钥
type APIKey struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	TenantID    string    `json:"tenant_id" gorm:"index"`
	UserID      string    `json:"user_id" gorm:"index"`
	Name        string    `json:"name"`
	KeyHash     string    `json:"key_hash"`    // SHA256(Key)
	KeyPrefix   string    `json:"key_prefix"`  // 前8位用于识别
	Permissions []Permission `json:"permissions" gorm:"serializer:json"`
	Models      []string  `json:"models" gorm:"serializer:json"` // 允许访问的模型
	RateLimit   int       `json:"rate_limit"`   // 每秒请求限制
	QuotaDaily  int64     `json:"quota_daily"`  // 每日Token配额
	Status      APIKeyStatus `json:"status"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
}

type APIKeyStatus string

const (
	APIKeyActive  APIKeyStatus = "active"
	APIKeyRevoked APIKeyStatus = "revoked"
	APIKeyExpired APIKeyStatus = "expired"
)

// HasPermission 检查角色是否拥有指定权限
func HasPermission(role Role, perm Permission) bool {
	perms, ok := RolePermissions[role]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == perm {
			return true
		}
	}
	return false
}

// HasAnyPermission 检查角色是否拥有任一权限
func HasAnyPermission(role Role, perms ...Permission) bool {
	for _, p := range perms {
		if HasPermission(role, p) {
			return true
		}
	}
	return false
}
