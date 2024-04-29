package http

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptrace"
	"os"
	"path/filepath"
	"time"

	"github.com/vadv/gopher-lua-libs/internal_matrics"
	lua "github.com/yuin/gopher-lua"
)

type luaRequest struct {
	*http.Request
}

type luaRequestStatistics struct {
	Start             time.Time
	ConnectStart      time.Time
	DnsStart          time.Time
	TlsHandshakeStart time.Time
	Ttfb              time.Duration
	DnsResolve        time.Duration
	ConnectTime       time.Duration
	TlsHandshake      time.Duration
	RequestWrote      time.Duration
	Trace             *httptrace.ClientTrace
}

const luaRequestType = "http_request_ud"
const luaRequestStatisitcsType = "http_request_statistics_ud"

func checkRequest(L *lua.LState, n int) *luaRequest {
	ud := L.CheckUserData(n)
	if v, ok := ud.Value.(*luaRequest); ok {
		return v
	}
	L.ArgError(n, "http request expected")
	return nil
}

func lvRequest(L *lua.LState, request *luaRequest) lua.LValue {
	ud := L.NewUserData()
	ud.Value = request
	L.SetMetatable(ud, L.GetTypeMetatable(luaRequestType))
	return ud
}

// http.request(verb, url, body) returns user-data, error
func NewRequest(L *lua.LState) int {
	verb := L.CheckString(1)
	url := L.CheckString(2)
	buffer := &bytes.Buffer{}
	if L.GetTop() > 2 {
		buffer.WriteString(L.CheckString(3))
		internal_matrics.MatAdd(L, "http.send", float64(buffer.Len()))
	}
	httpReq, err := http.NewRequest(verb, url, buffer)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	req := &luaRequest{Request: httpReq}
	req.Request.Header.Set(`User-Agent`, DefaultUserAgent)
	L.Push(lvRequest(L, req))
	return 1
}

func NewRequestStatistics() *luaRequestStatistics {

	statistic := &luaRequestStatistics{}
	statistic.Start = time.Now()

	statistic.Trace = &httptrace.ClientTrace{
		DNSStart: func(dsi httptrace.DNSStartInfo) { statistic.DnsStart = time.Now() },
		DNSDone: func(ddi httptrace.DNSDoneInfo) {
			statistic.DnsResolve = time.Since(statistic.DnsStart)
		},

		TLSHandshakeStart: func() { statistic.TlsHandshakeStart = time.Now() },
		TLSHandshakeDone: func(cs tls.ConnectionState, err error) {
			statistic.TlsHandshake = time.Since(statistic.TlsHandshakeStart)
		},

		ConnectStart: func(network, addr string) { statistic.ConnectStart = time.Now() },
		ConnectDone: func(network, addr string, err error) {
			statistic.ConnectTime = time.Since(statistic.ConnectStart)
		},

		GotFirstResponseByte: func() {
			statistic.Ttfb = time.Since(statistic.Start)
		},

		WroteRequest: func(wri httptrace.WroteRequestInfo) {
			statistic.RequestWrote = time.Since(statistic.Start)
		},
	}

	return statistic
}

func checkStatistics(L *lua.LState, n int) *luaRequestStatistics {
	ud := L.CheckUserData(n)
	if v, ok := ud.Value.(*luaRequestStatistics); ok {
		return v
	}
	L.ArgError(n, "http statistics expected")
	return nil
}

func lvStatistics(L *lua.LState, statistics *luaRequestStatistics) lua.LValue {
	ud := L.NewUserData()
	ud.Value = statistics
	L.SetMetatable(ud, L.GetTypeMetatable(luaRequestStatisitcsType))
	return ud
}

func AttachStatistics(L *lua.LState) int {
	req := checkRequest(L, 1)

	statistic := NewRequestStatistics()

	statisticReq := req.WithContext(httptrace.WithClientTrace(req.Context(), statistic.Trace))

	req = &luaRequest{Request: statisticReq}
	L.Push(lvRequest(L, req))
	L.Push(lvStatistics(L, statistic))
	return 2
}

func GetRequestStatistisc(L *lua.LState) int {
	statistics := checkStatistics(L, 1)

	statTable := L.NewTable()
	statTable.RawSetString("ttfb", lua.LNumber(statistics.Ttfb.Seconds()))
	statTable.RawSetString("dns", lua.LNumber(statistics.DnsResolve.Seconds()))
	statTable.RawSetString("connect", lua.LNumber(statistics.ConnectTime.Seconds()))
	statTable.RawSetString("wrote", lua.LNumber(statistics.RequestWrote.Seconds()))
	statTable.RawSetString("tls", lua.LNumber(statistics.TlsHandshake.Seconds()))

	L.Push(statTable)
	return 1
}

// http.filerequest(url, files, params) returns user-data, error
func NewFileRequest(L *lua.LState) int {
	url := L.CheckString(1)
	files := L.CheckTable(2)

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	var writeFile = func(info *lua.LTable, w *multipart.Writer) (err error) {
		fieldname := info.RawGetString("fieldname")
		path := info.RawGetString("path")
		if fieldname == lua.LNil || path == lua.LNil {
			return errors.New("fieldname or path is nil")
		}
		filename := info.RawGetString("filename")
		if filename == lua.LNil {
			filename = lua.LString(filepath.Base(path.String()))
		}

		part, err := writer.CreateFormFile(fieldname.String(), filename.String())
		if err != nil {
			return
		}
		file, err := os.Open(path.String())
		if err != nil {
			return
		}
		defer file.Close()
		_, err = io.Copy(part, file)
		return
	}

	var err error
	if files.Len() == 0 {
		err = writeFile(files, writer)
	} else {
		for key, value := files.Next(lua.LNil); key != lua.LNil; key, value = files.Next(key) {
			err = writeFile(value.(*lua.LTable), writer)
			if err != nil {
				break
			}
		}
	}

	if err == nil && L.GetTop() > 2 {
		fields := L.CheckTable(3)
		for key, value := fields.Next(lua.LNil); key != lua.LNil; key, value = fields.Next(key) {
			err = writer.WriteField(key.String(), value.String())
			if err != nil {
				break
			}
		}
	}

	if err == nil {
		err = writer.Close()
	}
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	httpReq, err := http.NewRequest("POST", url, body)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	req := &luaRequest{Request: httpReq}
	req.Request.Header.Set(`User-Agent`, DefaultUserAgent)
	req.Request.Header.Set(`Content-Type`, writer.FormDataContentType())
	L.Push(lvRequest(L, req))
	return 1
}

// request:set_basic_auth(username, password)
func SetBasicAuth(L *lua.LState) int {
	req := checkRequest(L, 1)
	user, passwd := L.CheckAny(2).String(), L.CheckAny(3).String()
	req.SetBasicAuth(user, passwd)
	return 0
}

func SetHost(L *lua.LState) int {
	req := checkRequest(L, 1)
	host := L.CheckAny(2).String()
	req.Host = host
	return 0
}

// request:header_set(key, value)
func HeaderSet(L *lua.LState) int {
	req := checkRequest(L, 1)
	key, value := L.CheckAny(2).String(), L.CheckAny(3).String()
	req.Header.Set(key, value)
	return 0
}

// DoRequest lua http_client_ud:do_request()
// http_client_ud:do_request(http_request_ud) returns (response, error)
//
//	response: {
//	  code = http_code (200, 201, ..., 500, ...),
//	  body = string
//	  headers = table
//	}
func DoRequest(L *lua.LState) int {
	internal_matrics.MatAdd(L, "http.req_num", 1)

	client := checkClient(L)
	req := checkRequest(L, 2)

	response, err := client.DoRequest(req.Request)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	defer response.Body.Close()
	headers := L.NewTable()
	for k, v := range response.Header {
		if len(v) > 0 {
			headers.RawSetString(k, lua.LString(v[0]))
		}
	}

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	internal_matrics.MatAdd(L, "http.receive", float64(len(data)))

	result := L.NewTable()
	L.SetField(result, `code`, lua.LNumber(response.StatusCode))
	L.SetField(result, `body`, lua.LString(string(data)))
	L.SetField(result, `headers`, headers)
	L.Push(result)
	return 1
}
