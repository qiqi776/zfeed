---@diagnostic disable: undefined-global
-- 原子更新评论对象缓存和列表索引
-- KEYS[1]=commentObjKey
-- KEYS[2]=commentIndexKey
-- ARGV[1]=obj_expire_seconds
-- ARGV[2]=index_expire_seconds
-- ARGV[3]=comment_id
-- ARGV[4]=serialized_comment_json

local objExpire = tonumber(ARGV[1]) or 0
local indexExpire = tonumber(ARGV[2]) or 0
local commentID = tonumber(ARGV[3])
local payload = ARGV[4]

if not commentID or not payload or payload == '' then
    return 0
end

if objExpire > 0 then
    redis.call('SETEX', KEYS[1], objExpire, payload)
else
    redis.call('SET', KEYS[1], payload)
end

redis.call('ZADD', KEYS[2], commentID, ARGV[3])
if indexExpire > 0 then
    redis.call('EXPIRE', KEYS[2], indexExpire)
end

return 1
