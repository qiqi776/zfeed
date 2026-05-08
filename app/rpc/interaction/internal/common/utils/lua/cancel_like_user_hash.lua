---@diagnostic disable: undefined-global
-- 用户维度取消点赞 HASH 脚本（TTL，原子性）
-- KEYS[1]=userLikeKey (like:user:{user_id})
-- ARGV[1]=cache_field
-- ARGV[2]=expire_seconds
-- 返回: {changed(0/1), existed(0/1)}

local expireTime = tonumber(ARGV[2]) or 0

local current = redis.call('HGET', KEYS[1], ARGV[1])
redis.call('HSET', KEYS[1], ARGV[1], '0')
if expireTime > 0 then
    redis.call('EXPIRE', KEYS[1], expireTime)
end

if current == '1' then
    return {1, 1}
end

if current == '0' then
    return {0, 1}
end

if current == false then
    return {0, 0}
end

return {0, 1}
