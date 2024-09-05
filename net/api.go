package lua_net

import (
	"context"
	"net"
	"time"

	"github.com/go-ping/ping"
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

func Ping(L *lua.LState) int {
	// ping
	domain := L.CheckString(1)
	if domain == "" {
		L.Push(L.NewTable())
		L.Push(lua.LString("no domoin or ip given"))
		return 2
	}

	count := L.CheckInt(2)
	if count < 1 {
		count = 1
	} else if count > 3 {
		count = 3
	}

	pinger, err := ping.NewPinger(domain)
	if err != nil {
		L.Push(L.NewTable())
		L.Push(lua.LString(err.Error()))
		return 2
	}

	pinger.SetPrivileged(true)
	pinger.Count = count
	pinger.Timeout = 5 * time.Second

	err = pinger.Run()
	if err != nil {
		L.Push(L.NewTable())
		L.Push(lua.LString(err.Error()))
		return 2
	}
	stats := pinger.Statistics()

	t := L.NewTable()

	rtts := L.NewTable()
	for _, t := range stats.Rtts {
		rtts.Append(lua.LNumber(t.Seconds()))
	}
	t.RawSetString("rtts", rtts)
	t.RawSetString("pkt_send", lua.LNumber(stats.PacketsSent))
	t.RawSetString("pkt_recv", lua.LNumber(stats.PacketsRecv))
	t.RawSetString("pkt_loss", lua.LNumber(stats.PacketLoss))
	t.RawSetString("pkt_recv_dup", lua.LNumber(stats.PacketsRecvDuplicates))
	t.RawSetString("ip", lua.LString(stats.IPAddr.String()))
	t.RawSetString("addr", lua.LString(stats.Addr))
	t.RawSetString("rtt_min", lua.LNumber(stats.MinRtt.Seconds()))
	t.RawSetString("rtt_avg", lua.LNumber(stats.AvgRtt.Seconds()))
	t.RawSetString("rtt_max", lua.LNumber(stats.MaxRtt.Seconds()))

	L.Push(t)
	L.Push(lua.LNil)
	return 2
}
