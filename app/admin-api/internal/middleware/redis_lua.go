package middleware

import _ "embed"

//go:embed verify_and_renew_session.lua
var VerifyAndRenewSessionScript string
