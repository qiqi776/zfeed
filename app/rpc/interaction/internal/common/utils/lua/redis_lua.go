package lua

import _ "embed"

//go:embed like_user_hash.lua
var LikeUserHashScript string

//go:embed cancel_like_user_hash.lua
var CancelLikeUserHashScript string
