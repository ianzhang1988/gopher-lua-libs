package internal_matrics

import (
	lua "github.com/yuin/gopher-lua"
)

// todo: use sync.Map

func GetMat(L *lua.LState) (map[string]float64, bool) {
	metric := L.GetGlobal("internal_matrics")
	metricUd, ok := metric.(*lua.LUserData)
	if !ok || metricUd.Value == nil {
		return nil, false
	}

	keyValue, ok := metricUd.Value.(map[string]float64)

	if !ok {
		return nil, false
	}

	return keyValue, true
}

func MatSet(L *lua.LState, name string, value float64) {
	if keyValue, ok := GetMat(L); ok {
		keyValue[name] = value
	}
}

func MatAdd(L *lua.LState, name string, value float64) {
	if keyValue, ok := GetMat(L); ok {
		if oldValue, ok := keyValue[name]; ok {
			keyValue[name] = oldValue + value
		} else {
			keyValue[name] = value
		}
	}
}

func Preload(L *lua.LState) {
	metric := map[string]float64{}
	MetricUD := L.NewUserData()
	MetricUD.Value = metric

	L.SetGlobal("internal_matrics", MetricUD)
}
