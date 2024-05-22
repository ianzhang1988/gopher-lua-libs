// Package http implements golang package http functionality for lua.

package lua_net

import (
	lua "github.com/yuin/gopher-lua"
)

// Preload adds http to the given Lua state's package.preload table. After it
// has been preloaded, it can be loaded using require:
//
//	local http = require("http")
func Preload(L *lua.LState) {
	L.PreloadModule("net", Loader)
}

// Loader is the module loader function.
func Loader(L *lua.LState) int {
	t := L.NewTable()
	L.SetFuncs(t, api)
	L.Push(t)
	return 1
}

var api = map[string]lua.LGFunction{
	"dnslookup": DnsLookup,
}
