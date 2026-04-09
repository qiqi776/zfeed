package lua

import _ "embed"

//go:embed query_follow_inbox_zset.lua
var QueryFollowInboxZSetScript string

//go:embed query_user_publish_zset.lua
var QueryUserPublishZSetScript string

//go:embed query_user_favorite_zset.lua
var QueryUserFavoriteZSetScript string

//go:embed update_user_publish_zset.lua
var UpdateUserPublishZSetScript string

//go:embed backfill_follow_inbox_zset.lua
var BackfillFollowInboxZSetScript string

//go:embed update_follow_inbox_zset.lua
var UpdateFollowInboxZSetScript string
