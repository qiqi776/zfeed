---@diagnostic disable: undefined-global
-- 用户维度取消点赞 HASH 脚本（TTL，原子性）
-- KEYS[1]=userLikeKey (like:user:{user_id})
-- ARGV[1]=content_id
-- ARGV[2]=expire_seconds
-- 返回: {changed(0/1), existed(0/1)}

local expireTime = tonumber(ARGV[2]) or 0

local removed = redis.call('HDEL', KEYS[1], ARGV[1])
if expireTime > 0 then
    redis.call('EXPIRE', KEYS[1], expireTime)
end

if removed == 0 then
    return {0, 0}
end

return {1, 1}
