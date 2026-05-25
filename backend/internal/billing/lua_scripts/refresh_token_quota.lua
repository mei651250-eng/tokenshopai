-- refresh_token_quota.lua
-- 令牌桶限流 + 配额扣减
-- KEYS[1]: 令牌桶 rate limit key (ratelimit:{tenant_id}:{api_key_id})
-- KEYS[2]: 配额 key (quota:{tenant_id}:{model_name}:{period})
-- ARGV[1]: 令牌桶容量
-- ARGV[2]: 令牌填充速率（个/秒）
-- ARGV[3]: 当前时间戳（微秒）
-- ARGV[4]: 请求数量
-- ARGV[5]: 配额上限
-- ARGV[6]: 配额周期（秒）
--
-- 返回:
--   >=0  剩余令牌数
--   -1   限流
--   -2   配额超限

-- 令牌桶限流
local rateKey = KEYS[1]
local capacity = tonumber(ARGV[1])
local rate = tonumber(ARGV[2])
local now = tonumber(ARGV[3]) / 1000000 -- 转为秒
local requested = tonumber(ARGV[4])

local tokens = tonumber(redis.call("GET", rateKey) or tostring(capacity))
local lastTime = tonumber(redis.call("GET", rateKey .. ":ts") or "0")

-- 补充令牌
if lastTime > 0 then
    local elapsed = now - lastTime
    tokens = math.min(capacity, tokens + elapsed * rate)
end

-- 检查令牌数
if tokens < requested then
    return -1
end

-- 消耗令牌
tokens = tokens - requested
redis.call("SET", rateKey, tokens)
redis.call("SET", rateKey .. ":ts", now)

-- 配额检查
local quotaKey = KEYS[2]
local quotaLimit = tonumber(ARGV[5])
local quotaPeriod = tonumber(ARGV[6])

local currentQuota = tonumber(redis.call("GET", quotaKey) or "0")
if currentQuota + requested > quotaLimit then
    return -2
end

-- 增加配额使用
redis.call("INCRBY", quotaKey, requested)
-- 首次设置过期时间
if currentQuota == 0 then
    redis.call("EXPIRE", quotaKey, quotaPeriod)
end

return math.floor(tokens)
