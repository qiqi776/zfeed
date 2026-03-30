package lua

import _ "embed"

//go:embed save_session.lua
var SaveSessionScript string

//go:embed remove_session.lua
var RemoveSessionScript string