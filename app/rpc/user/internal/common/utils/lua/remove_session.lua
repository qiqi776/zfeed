---@diagnostic disable: undefined-global

local curToken = redis.call("GET", KEYS[2])
if curToken == ARGV[1] then
  redis.call("DEL", KEYS[2])
  redis.call("DEL", KEYS[1])
end
return 1
