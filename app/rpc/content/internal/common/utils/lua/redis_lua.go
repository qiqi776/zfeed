package lua

import _ "embed"

//go:embed query_hot_feed_zset.lua
var QueryHotFeedZSetScript string

//go:embed merge_hot_inc.lua
var MergeHotIncScript string

//go:embed rebuild_hot_feed_zset.lua
var RebuildHotFeedZSetScript string

//go:embed rebuild_hot_snapshot.lua
var RebuildHotSnapshotScript string

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
