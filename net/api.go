package lua_net

import (
	"context"
	"net"
	"time"

	lua "github.com/yuin/gopher-lua"
)

func DnsLookup(L *lua.LState) int {
	// 解析域名
	domain := L.CheckString(1)
	if domain == "" {
		L.Push(L.NewTable())
		L.Push(lua.LString("no domoin given"))
		return 2
	}

	timeout := 1
	v := L.Get(2)
	if intv, ok := v.(lua.LNumber); ok {
		timeout = int(intv)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	// addrs, err := net.LookupHost(domain)
	addrs, err := net.DefaultResolver.LookupHost(ctx, domain)
	if err != nil {
		L.Push(L.NewTable())
		L.Push(lua.LString(err.Error()))
		return 2
	}

	t := L.NewTable()
	for _, addr := range addrs {
		t.Append(lua.LString(addr))
	}
	L.Push(t)
	L.Push(lua.LNil)
	return 2
}
