---@diagnostic disable: undefined-global
-- KEYS[1] = inbox zset key
-- ARGV[1...] = content_id members to remove
-- return: removed member count

local key = KEYS[1]
local removed = 0

for i = 1, #ARGV do
  local member = ARGV[i]
  if member ~= nil and member ~= '' then
    removed = removed + redis.call('ZREM', key, member)
  end
end

return removed
