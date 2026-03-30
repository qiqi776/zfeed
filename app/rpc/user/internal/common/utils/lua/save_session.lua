---@diagnostic disable: undefined-global

local oldToken = redis.call("GET", KEYS[2])
if oldToken and oldToken ~= "" then
  redis.call("DEL", ARGV[4] .. ":" .. oldToken)
end
redis.call("SETEX", KEYS[1], ARGV[3], ARGV[1])
redis.call("SETEX", KEYS[2], ARGV[3], ARGV[2])
return 1
