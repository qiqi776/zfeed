package lua

import _ "embed"

//go:embed like_user_hash.lua
var LikeUserHashScript string

//go:embed cancel_like_user_hash.lua
var CancelLikeUserHashScript string

//go:embed update_comment_cache.lua
var UpdateCommentCacheScript string

//go:embed batch_get_comment_objs.lua
var BatchGetCommentObjsScript string

//go:embed add_user_favorite_if_exists.lua
var AddUserFavoriteIfExistsScript string
