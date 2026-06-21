---@diagnostic disable: undefined-global

local userId = redis.call("GET", KEYS[1])
if not userId or userId == "" then
  return ""
end

local userKey = ARGV[2] .. ":" .. userId
local token = redis.call("GET", userKey)
if not token or token == "" or token ~= ARGV[1] then
  return ""
end

local ttl = redis.call("TTL", KEYS[1])
if ttl and ttl >= 0 and ttl < tonumber(ARGV[4]) then
  redis.call("EXPIRE", KEYS[1], tonumber(ARGV[3]))
  redis.call("EXPIRE", userKey, tonumber(ARGV[3]))
end

return userId
