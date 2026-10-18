package paymentScripts

import (
	_ "embed"
	"github.com/redis/go-redis/v9"
)

//go:embed paymentAuthorizeScript.lua
var paymentAuthorizeScriptLua string

var PaymentAuthorizeScript = redis.NewScript(paymentAuthorizeScriptLua)
