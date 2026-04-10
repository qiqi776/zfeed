---@diagnostic disable: undefined-global
-- KEYS[1] = increment hash key
-- KEYS[2] = global hot zset key
-- ARGV[1] = precision
-- ARGV[2...] = member, delta, member, delta ...
-- return: {merged_count}

local incKey = KEYS[1]
local zsetKey = KEYS[2]
local precision = tonumber(ARGV[1]) or 0

local items = nil
if ARGV ~= nil and #ARGV >= 3 and ((#ARGV - 1) % 2 == 0) then
    items = {}
    for i = 2, #ARGV do
        table.insert(items, ARGV[i])
    end
else
    items = redis.call('HGETALL', incKey)
end

local merged = 0
if items ~= nil and #items > 0 then
    for i = 1, #items, 2 do
        local member = items[i]
        local delta = tonumber(items[i + 1])
        if member ~= nil and member ~= '' and delta ~= nil and delta ~= 0 then
            if precision > 0 then
                local factor = 10 ^ precision
                delta = math.floor(delta * factor + 0.5) / factor
            end
            redis.call('ZINCRBY', zsetKey, delta, member)
            merged = merged + 1
        end
    end
end

redis.call('DEL', incKey)
return {merged}

