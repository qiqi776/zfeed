---@diagnostic disable: undefined-global
-- 批量读取评论对象缓存
-- KEYS = comment:item:{comment_id} ...

if #KEYS == 0 then
    return {}
end

return redis.call('MGET', unpack(KEYS))
