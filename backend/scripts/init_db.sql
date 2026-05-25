-- ============================================================
-- TokenHub 数据库初始化脚本
-- PostgreSQL 15+
-- 执行方式: psql -U tokenhub -d tokenhub -f init_db.sql
-- ============================================================

-- 1. 扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- 2. 核心表结构
-- ============================================================

-- 租户表
CREATE TABLE IF NOT EXISTS tenants (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(50) UNIQUE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'trial',
    plan VARCHAR(20) NOT NULL DEFAULT 'free',
    region VARCHAR(10) NOT NULL DEFAULT 'cn',
    language VARCHAR(10) NOT NULL DEFAULT 'zh-CN',
    currency VARCHAR(10) NOT NULL DEFAULT 'CNY',
    timezone VARCHAR(50) NOT NULL DEFAULT 'Asia/Shanghai',
    max_users INT NOT NULL DEFAULT 5,
    max_api_keys INT NOT NULL DEFAULT 10,
    max_models INT NOT NULL DEFAULT 20,
    max_qps INT NOT NULL DEFAULT 100,
    isolation VARCHAR(20) NOT NULL DEFAULT 'logical',
    config JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ
);

CREATE INDEX idx_tenants_slug ON tenants(slug);
CREATE INDEX idx_tenants_status ON tenants(status);

-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(36) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(100) NOT NULL,
    phone VARCHAR(20),
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(30) NOT NULL DEFAULT 'developer',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    department_id VARCHAR(36),
    language VARCHAR(10) NOT NULL DEFAULT 'zh-CN',
    timezone VARCHAR(50) NOT NULL DEFAULT 'Asia/Shanghai',
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, email)
);

CREATE INDEX idx_users_tenant ON users(tenant_id);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_status ON users(status);

-- API密钥表
CREATE TABLE IF NOT EXISTS api_keys (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(36) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    key_hash VARCHAR(64) NOT NULL,
    key_prefix VARCHAR(16) NOT NULL,
    permissions JSONB NOT NULL DEFAULT '[]',
    models JSONB NOT NULL DEFAULT '[]',
    rate_limit INT NOT NULL DEFAULT 10,
    quota_daily BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ
);

CREATE INDEX idx_api_keys_tenant ON api_keys(tenant_id);
CREATE INDEX idx_api_keys_user ON api_keys(user_id);
CREATE INDEX idx_api_keys_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_prefix ON api_keys(key_prefix);
CREATE INDEX idx_api_keys_status ON api_keys(status);

-- 人脸识别凭据表
CREATE TABLE IF NOT EXISTS face_credentials (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id TEXT NOT NULL,
    public_key TEXT NOT NULL,
    sign_count INT NOT NULL DEFAULT 0,
    transport JSONB NOT NULL DEFAULT '[]',
    aa_guid VARCHAR(50),
    name VARCHAR(100),
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, credential_id)
);

-- 钱包绑定表
CREATE TABLE IF NOT EXISTS wallet_bindings (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id VARCHAR(36) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    wallet_type VARCHAR(30) NOT NULL,
    address VARCHAR(255) NOT NULL,
    chain_type VARCHAR(30) NOT NULL,
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    bind_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    unbind_at TIMESTAMPTZ,
    label VARCHAR(100)
);

CREATE INDEX idx_wallet_bindings_user ON wallet_bindings(user_id);
CREATE INDEX idx_wallet_bindings_address ON wallet_bindings(address);
CREATE UNIQUE INDEX idx_wallet_bindings_unique ON wallet_bindings(user_id, address, chain_type) WHERE unbind_at IS NULL;

-- 加密货币充值订单表
CREATE TABLE IF NOT EXISTS crypto_deposit_orders (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id VARCHAR(36) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    order_no VARCHAR(32) UNIQUE NOT NULL,
    currency VARCHAR(10) NOT NULL,
    chain_type VARCHAR(30) NOT NULL,
    amount DECIMAL(20, 8) NOT NULL,
    to_address VARCHAR(255) NOT NULL,
    from_address VARCHAR(255),
    tx_hash VARCHAR(255),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    confirmations INT NOT NULL DEFAULT 0,
    exchange_rate DECIMAL(20, 8),
    fiat_amount DECIMAL(20, 2),
    fiat_currency VARCHAR(10),
    expired_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_crypto_deposit_user ON crypto_deposit_orders(user_id);
CREATE INDEX idx_crypto_deposit_order ON crypto_deposit_orders(order_no);
CREATE INDEX idx_crypto_deposit_status ON crypto_deposit_orders(status);

-- 支付订单表
CREATE TABLE IF NOT EXISTS payment_orders (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id VARCHAR(36) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    order_no VARCHAR(32) UNIQUE NOT NULL,
    channel VARCHAR(30) NOT NULL,
    amount BIGINT NOT NULL,
    currency VARCHAR(10) NOT NULL DEFAULT 'CNY',
    fee_amount BIGINT NOT NULL DEFAULT 0,
    actual_amount BIGINT NOT NULL DEFAULT 0,
    to_currency VARCHAR(10) NOT NULL DEFAULT 'CNY',
    exchange_rate DECIMAL(20, 8),
    status VARCHAR(20) NOT NULL DEFAULT 'created',
    redirect_url TEXT,
    qr_code TEXT,
    expired_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payment_orders_user ON payment_orders(user_id);
CREATE INDEX idx_payment_orders_order ON payment_orders(order_no);
CREATE INDEX idx_payment_orders_status ON payment_orders(status);

-- 计费记录表
CREATE TABLE IF NOT EXISTS billing_records (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    api_key_id VARCHAR(36),
    model_id VARCHAR(50),
    model_name VARCHAR(100),
    provider VARCHAR(30),
    trace_id VARCHAR(36),
    input_tokens INT NOT NULL DEFAULT 0,
    output_tokens INT NOT NULL DEFAULT 0,
    total_tokens INT NOT NULL DEFAULT 0,
    amount BIGINT NOT NULL DEFAULT 0,
    currency VARCHAR(10) NOT NULL DEFAULT 'CNY',
    balance_before BIGINT NOT NULL DEFAULT 0,
    balance_after BIGINT NOT NULL DEFAULT 0,
    billing_type VARCHAR(20) NOT NULL DEFAULT 'pay_as_you_go',
    package_id VARCHAR(36),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_billing_records_tenant ON billing_records(tenant_id);
CREATE INDEX idx_billing_records_user ON billing_records(user_id);
CREATE INDEX idx_billing_records_time ON billing_records(created_at);
CREATE INDEX idx_billing_records_model ON billing_records(model_name);

-- 收款账号表
CREATE TABLE IF NOT EXISTS receiving_accounts (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id VARCHAR(36) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    account_type VARCHAR(20) NOT NULL,
    account_name VARCHAR(100) NOT NULL,
    account_no VARCHAR(100),
    bank_name VARCHAR(100),
    bank_branch VARCHAR(200),
    qrcode_url TEXT,
    wallet_address VARCHAR(255),
    chain_type VARCHAR(30),
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    is_verified BOOLEAN NOT NULL DEFAULT FALSE,
    label VARCHAR(100),
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_receiving_accounts_user ON receiving_accounts(user_id);

-- 提现账号表
CREATE TABLE IF NOT EXISTS withdraw_accounts (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id VARCHAR(36) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    account_type VARCHAR(20) NOT NULL,
    account_name VARCHAR(100) NOT NULL,
    account_no VARCHAR(100) NOT NULL,
    bank_name VARCHAR(100),
    bank_branch VARCHAR(200),
    wallet_address VARCHAR(255),
    chain_type VARCHAR(30),
    is_verified BOOLEAN NOT NULL DEFAULT FALSE,
    label VARCHAR(100),
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_withdraw_accounts_user ON withdraw_accounts(user_id);

-- 提现订单表
CREATE TABLE IF NOT EXISTS withdrawal_orders (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id VARCHAR(36) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    order_no VARCHAR(32) UNIQUE NOT NULL,
    amount BIGINT NOT NULL,
    currency VARCHAR(10) NOT NULL DEFAULT 'CNY',
    fee BIGINT NOT NULL DEFAULT 0,
    actual_amount BIGINT NOT NULL DEFAULT 0,
    withdraw_account_id VARCHAR(36) NOT NULL REFERENCES withdraw_accounts(id),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    reject_reason TEXT,
    reviewed_by VARCHAR(36),
    reviewed_at TIMESTAMPTZ,
    transferred_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_withdrawal_orders_user ON withdrawal_orders(user_id);
CREATE INDEX idx_withdrawal_orders_status ON withdrawal_orders(status);

-- 审计日志表
CREATE TABLE IF NOT EXISTS audit_logs (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(36),
    user_id VARCHAR(36),
    action VARCHAR(100) NOT NULL,
    resource VARCHAR(100),
    resource_id VARCHAR(36),
    detail JSONB,
    ip VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_tenant ON audit_logs(tenant_id);
CREATE INDEX idx_audit_logs_user ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_time ON audit_logs(created_at);

-- 通知表
CREATE TABLE IF NOT EXISTS notifications (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(36),
    user_id VARCHAR(36),
    title VARCHAR(200) NOT NULL,
    message TEXT NOT NULL,
    level VARCHAR(20) NOT NULL DEFAULT 'info',
    type VARCHAR(30),
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    link VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_user ON notifications(user_id);
CREATE INDEX idx_notifications_read ON notifications(user_id, is_read);
CREATE INDEX idx_notifications_time ON notifications(created_at);

-- RBAC 角色定义表
CREATE TABLE IF NOT EXISTS roles (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(36) REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,
    display_name VARCHAR(100),
    permissions JSONB NOT NULL DEFAULT '[]',
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

-- 模型配置表
CREATE TABLE IF NOT EXISTS model_configs (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(36) REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    display_name VARCHAR(100),
    provider VARCHAR(30) NOT NULL,
    model_id VARCHAR(100) NOT NULL,
    endpoint_url TEXT NOT NULL,
    api_key_enc TEXT NOT NULL,
    input_price DECIMAL(20, 8) NOT NULL DEFAULT 0,
    output_price DECIMAL(20, 8) NOT NULL DEFAULT 0,
    currency VARCHAR(10) NOT NULL DEFAULT 'CNY',
    max_tokens INT NOT NULL DEFAULT 4096,
    weight INT NOT NULL DEFAULT 100,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    circuit_state VARCHAR(20) NOT NULL DEFAULT 'closed',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_model_configs_tenant ON model_configs(tenant_id);
CREATE INDEX idx_model_configs_provider ON model_configs(provider);

-- 租户配额表
CREATE TABLE IF NOT EXISTS tenant_quotas (
    tenant_id VARCHAR(36) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    model_name VARCHAR(100) NOT NULL,
    daily_tokens BIGINT NOT NULL DEFAULT 0,
    monthly_tokens BIGINT NOT NULL DEFAULT 0,
    used_daily_tokens BIGINT NOT NULL DEFAULT 0,
    used_monthly_tokens BIGINT NOT NULL DEFAULT 0,
    period_start TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, model_name)
);

-- 退款订单表
CREATE TABLE IF NOT EXISTS refund_orders (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id VARCHAR(36) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    order_no VARCHAR(32) UNIQUE NOT NULL,
    payment_order_no VARCHAR(32) NOT NULL,
    amount BIGINT NOT NULL,
    currency VARCHAR(10) NOT NULL DEFAULT 'CNY',
    reason TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    channel_refund_no VARCHAR(100),
    reject_reason TEXT,
    reviewed_by VARCHAR(36),
    reviewed_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refund_orders_user ON refund_orders(user_id);
CREATE INDEX idx_refund_orders_payment ON refund_orders(payment_order_no);
CREATE INDEX idx_refund_orders_status ON refund_orders(status);

-- 对账汇总表
CREATE TABLE IF NOT EXISTS reconciliation_summaries (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    date DATE NOT NULL,
    tenant_id VARCHAR(36),
    payment_count BIGINT NOT NULL DEFAULT 0,
    payment_amount BIGINT NOT NULL DEFAULT 0,
    payment_fee BIGINT NOT NULL DEFAULT 0,
    actual_income BIGINT NOT NULL DEFAULT 0,
    api_call_count BIGINT NOT NULL DEFAULT 0,
    token_consumed BIGINT NOT NULL DEFAULT 0,
    api_cost BIGINT NOT NULL DEFAULT 0,
    withdrawal_count BIGINT NOT NULL DEFAULT 0,
    withdrawal_amount BIGINT NOT NULL DEFAULT 0,
    refund_count BIGINT NOT NULL DEFAULT 0,
    refund_amount BIGINT NOT NULL DEFAULT 0,
    gross_profit BIGINT NOT NULL DEFAULT 0,
    net_profit BIGINT NOT NULL DEFAULT 0,
    recon_status VARCHAR(20) NOT NULL DEFAULT 'matched',
    discrepancy BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(date, tenant_id)
);

CREATE INDEX idx_recon_summaries_date ON reconciliation_summaries(date);
CREATE INDEX idx_recon_summaries_tenant ON reconciliation_summaries(tenant_id);

-- 分销商表
CREATE TABLE IF NOT EXISTS distributors (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id VARCHAR(36) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    role VARCHAR(30) NOT NULL DEFAULT 'agent',
    referral_code VARCHAR(20) UNIQUE NOT NULL,
    parent_id VARCHAR(36),
    level INT NOT NULL DEFAULT 1,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    commission_type VARCHAR(20) NOT NULL DEFAULT 'percent',
    commission_rate DECIMAL(10, 4) NOT NULL DEFAULT 0.15,
    total_referred INT NOT NULL DEFAULT 0,
    total_revenue BIGINT NOT NULL DEFAULT 0,
    total_commission BIGINT NOT NULL DEFAULT 0,
    pending_commission BIGINT NOT NULL DEFAULT 0,
    withdrawn_commission BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_distributors_user ON distributors(user_id);
CREATE INDEX idx_distributors_tenant ON distributors(tenant_id);
CREATE INDEX idx_distributors_referral ON distributors(referral_code);
CREATE INDEX idx_distributors_parent ON distributors(parent_id);

-- 佣金记录表
CREATE TABLE IF NOT EXISTS commission_records (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    distributor_id VARCHAR(36) NOT NULL REFERENCES distributors(id) ON DELETE CASCADE,
    referral_user_id VARCHAR(36) NOT NULL,
    order_no VARCHAR(32) NOT NULL,
    order_amount BIGINT NOT NULL,
    commission_rate DECIMAL(10, 4) NOT NULL,
    commission_amt BIGINT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    period VARCHAR(7) NOT NULL,
    settled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_commission_records_dist ON commission_records(distributor_id);
CREATE INDEX idx_commission_records_period ON commission_records(period);
CREATE INDEX idx_commission_records_status ON commission_records(status);

-- 告警规则表
CREATE TABLE IF NOT EXISTS alert_rules (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    metric VARCHAR(50) NOT NULL,
    condition VARCHAR(10) NOT NULL DEFAULT 'gt',
    threshold DECIMAL(20, 4) NOT NULL,
    severity VARCHAR(20) NOT NULL DEFAULT 'warning',
    cooldown INT NOT NULL DEFAULT 300,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 插入默认告警规则
INSERT INTO alert_rules (id, name, metric, condition, threshold, severity, cooldown) VALUES
('alert-001', 'API错误率过高', 'error_rate', 'gt', 5.0, 'warning', 600),
('alert-002', 'API错误率严重', 'error_rate', 'gt', 10.0, 'critical', 300),
('alert-003', 'P99延迟过高', 'latency_p99', 'gt', 10000, 'warning', 600),
('alert-004', '余额不足', 'balance', 'lt', 1000, 'critical', 300)
ON CONFLICT DO NOTHING;

-- ============================================================
-- 3. 初始数据（种子数据）
-- ============================================================

-- 默认租户
INSERT INTO tenants (id, name, slug, status, plan, region, max_users, max_api_keys, max_models, max_qps, config)
VALUES (
    'tenant-default-00000000-000000000001',
    '默认租户',
    'default',
    'active',
    'enterprise',
    'cn',
    100, 50, 200, 10000,
    '{"custom_branding": false, "enable_audit": true, "enable_desensitize": true, "rate_limit_rps": 100}'
) ON CONFLICT (slug) DO NOTHING;

-- 超级管理员 (密码: Admin@2026)
INSERT INTO users (id, tenant_id, username, email, phone, password_hash, role, status)
VALUES (
    'user-superadmin-00000000-000000000001',
    'tenant-default-00000000-000000000001',
    'superadmin',
    'admin@tokenhub.com',
    '13800000000',
    '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
    'super_admin',
    'active'
) ON CONFLICT (tenant_id, email) DO NOTHING;

-- 演示租户
INSERT INTO tenants (id, name, slug, status, plan, region, max_users, max_api_keys, max_models, max_qps, config)
VALUES (
    'tenant-demo-00000000-00000000-000000000002',
    '演示租户',
    'demo',
    'trial',
    'starter',
    'cn',
    5, 3, 10, 50,
    '{"custom_branding": false, "enable_audit": true, "enable_desensitize": false, "rate_limit_rps": 20}'
) ON CONFLICT (slug) DO NOTHING;

-- 演示租户管理员 (密码: Demo@2026)
INSERT INTO users (id, tenant_id, username, email, password_hash, role, status)
VALUES (
    'user-demo-admin-00000000-00000000-000000000002',
    'tenant-demo-00000000-00000000-000000000002',
    'demo_admin',
    'demo@tokenhub.com',
    '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
    'tenant_admin',
    'active'
) ON CONFLICT (tenant_id, email) DO NOTHING;

-- 系统角色
INSERT INTO roles (id, tenant_id, name, display_name, permissions, is_system) VALUES
('role-super-admin', NULL, 'super_admin', '超级管理员', '["*"]', TRUE),
('role-tenant-admin', NULL, 'tenant_admin', '租户管理员', '["model:list","model:create","model:update","model:route","apikey:list","apikey:create","apikey:revoke","billing:view","billing:topup","billing:export","user:create","user:update","user:delete","monitor:view","monitor:export","security:view","report:view","report:export"]', TRUE),
('role-dept-admin', NULL, 'dept_admin', '部门管理员', '["model:list","model:route","apikey:list","apikey:create","billing:view","user:create","user:update","monitor:view","report:view"]', TRUE),
('role-developer', NULL, 'developer', '开发者', '["model:list","model:route","apikey:list","apikey:create","billing:view","monitor:view"]', TRUE),
('role-viewer', NULL, 'viewer', '只读用户', '["model:list","billing:view","monitor:view","report:view"]', TRUE),
('role-api-consumer', NULL, 'api_consumer', 'API消费者', '["model:route"]', TRUE)
ON CONFLICT (tenant_id, name) DO NOTHING;

-- 默认模型配置
INSERT INTO model_configs (id, tenant_id, name, display_name, provider, model_id, endpoint_url, api_key_enc, input_price, output_price, currency, max_tokens, weight) VALUES
('model-gpt4o-001', 'tenant-default-00000000-000000000001', 'gpt-4o', 'GPT-4o', 'openai', 'gpt-4o', 'https://api.openai.com/v1', '', 0.005, 0.015, 'USD', 128000, 100),
('model-claude35-001', 'tenant-default-00000000-000000000001', 'claude-3.5-sonnet', 'Claude 3.5 Sonnet', 'anthropic', 'claude-3-5-sonnet-20241022', 'https://api.anthropic.com/v1', '', 0.003, 0.015, 'USD', 200000, 90),
('model-qwen-max-001', 'tenant-default-00000000-000000000001', 'qwen-max', '通义千问-Max', 'alibaba', 'qwen-max', 'https://dashscope.aliyuncs.com/compatible-mode/v1', '', 0.004, 0.008, 'CNY', 32768, 80),
('model-deepseek-001', 'tenant-default-00000000-000000000001', 'deepseek-v2', 'DeepSeek-V2', 'deepseek', 'deepseek-chat', 'https://api.deepseek.com/v1', '', 0.001, 0.002, 'CNY', 65536, 70),
('model-glm4-001', 'tenant-default-00000000-000000000001', 'glm-4', 'GLM-4', 'zhipu', 'glm-4', 'https://open.bigmodel.cn/api/paas/v4', '', 0.005, 0.005, 'CNY', 128000, 60)
ON CONFLICT DO NOTHING;

-- ============================================================
-- 4. 更新触发器（自动维护 updated_at）
-- ============================================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 为所有含 updated_at 的表创建触发器
DO $$
DECLARE
    t TEXT;
BEGIN
    FOR t IN
        SELECT table_name FROM information_schema.columns
        WHERE column_name = 'updated_at' AND table_schema = 'public'
    LOOP
        EXECUTE format('
            DROP TRIGGER IF EXISTS set_updated_at ON %I;
            CREATE TRIGGER set_updated_at
                BEFORE UPDATE ON %I
                FOR EACH ROW
                EXECUTE FUNCTION update_updated_at_column();
        ', t, t);
    END LOOP;
END;
$$;
