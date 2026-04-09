---@diagnostic disable: undefined-global
-- KEYS[1] = user favorite zset key
-- ARGV[1] = cursor score (favorite_id string), empty/"0" means first page
-- ARGV[2] = page size
-- return: {exists, has_more, next_cursor, content_id1, content_id2, ...}

local key = KEYS[1]
local cursor = ARGV[1]
local pageSize = tonumber(ARGV[2])

local exists = redis.call('EXISTS', key)
if exists == 0 then
  return {0, 0, ""}
end

if pageSize == nil or pageSize <= 0 then
  return {1, 0, ""}
end

local maxScore = "+inf"
if cursor ~= nil and cursor ~= "" and cursor ~= "0" then
  maxScore = cursor
end

local entries = redis.call('ZREVRANGEBYSCORE', key, '(' .. maxScore, '-inf', 'WITHSCORES', 'LIMIT', 0, pageSize + 1)
local count = math.floor(#entries / 2)

local hasMore = 0
if count > pageSize then
  hasMore = 1
end

local nextCursor = ""
if hasMore == 1 then
  nextCursor = entries[pageSize * 2]
end

local res = {1, hasMore, nextCursor}
for i = 1, math.min(count, pageSize) do
  res[#res + 1] = entries[(i - 1) * 2 + 1]
end

return res
