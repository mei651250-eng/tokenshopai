-- deduct_balance.lua
-- Redis + Lua 原子扣费脚本
-- KEYS[1]: 用户余额 key (balance:{tenant_id}:{user_id})
-- KEYS[2]: 用户冻结余额 key (frozen:{tenant_id}:{user_id})
-- ARGV[1]: 扣费金额（分）
-- ARGV[2]: 交易ID（幂等）
-- ARGV[3]: 最低余额限制（分）
-- ARGV[4]: 幂等key (idempotent:{tx_id})
-- 
-- 返回:
--   1  扣费成功
--  -1  余额不足
--  -2  重复交易（已扣费）
--  -3  参数错误

-- 幂等检查
local txKey = ARGV[4]
if redis.call("EXISTS", txKey) == 1 then
    return -2
end

local balanceKey = KEYS[1]
local amount = tonumber(ARGV[1])
local minBalance = tonumber(ARGV[3])

if amount == nil or amount <= 0 then
    return -3
end

-- 获取当前余额
local currentBalance = tonumber(redis.call("GET", balanceKey) or "0")

-- 检查余额是否充足（扣费后不低于最低余额）
if currentBalance - amount < minBalance then
    return -1
end

-- 原子扣减
redis.call("DECRBY", balanceKey, amount)

-- 记录幂等标记，设置24小时过期
redis.call("SET", txKey, "1", "EX", 86400)

-- 记录扣费流水（List，保留最近1000条）
local logKey = "txlog:" .. string.match(balanceKey, "balance:(.+)")
local logEntry = ARGV[2] .. ":" .. amount .. ":" .. currentBalance .. ":" .. (currentBalance - amount) .. ":" .. tostring(redis.call("TIME")[1])
redis.call("RPUSH", logKey, logEntry)
redis.call("LTRIM", logKey, -1000, -1)

return 1
