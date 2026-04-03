---@diagnostic disable: undefined-global
-- 用户维度点赞 HASH 写入脚本（TTL，原子性）
-- KEYS[1]=userLikeKey (like:user:{user_id})
-- ARGV[1]=content_id
-- ARGV[2]=expire_seconds
-- 返回: {changed(0/1), cached(0/1)}
--   changed: 1=本次确实从未点赞->已点赞；0=重复点赞
--   cached: 1=写入了缓存

local cid = tonumber(ARGV[1])
local expireTime = tonumber(ARGV[2]) or 0

if not cid then
    return {0, 0}
end

local added = redis.call('HSETNX', KEYS[1], ARGV[1], '1')
if expireTime > 0 then
    redis.call('EXPIRE', KEYS[1], expireTime)
end

if added == 0 then
    return {0, 1}
end

return {1, 1}
