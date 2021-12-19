package metadata

import (
	"fmt"
	"strconv"
	"time"
)

type Metadata interface {
	IsExists(key string) bool
	Set(key string, value interface{})
	Get(key string) interface{}
	GetBool(key string) bool
	GetInt(key string) int
	GetFloat(key string) float64
	GetDuration(key string) time.Duration
	GetString(key string) string
	GetStrings(key string) []string
}

type MapMetadata map[string]interface{}

func (m MapMetadata) IsExists(key string) bool {
	_, ok := m[key]
	return ok
}

func (m MapMetadata) Set(key string, value interface{}) {
	m[key] = value
}

func (m MapMetadata) Get(key string) interface{} {
	if m != nil {
		return m[key]
	}
	return nil
}

func (m MapMetadata) GetBool(key string) (v bool) {
	if m == nil || !m.IsExists(key) {
		return
	}
	switch vv := m[key].(type) {
	case bool:
		return vv
	case int:
		return vv != 0
	case string:
		v, _ = strconv.ParseBool(vv)
		return
	}
	return
}

func (m MapMetadata) GetInt(key string) (v int) {
	switch vv := m[key].(type) {
	case bool:
		if vv {
			v = 1
		}
	case int:
		return vv
	case string:
		v, _ = strconv.Atoi(vv)
		return
	}
	return
}

func (m MapMetadata) GetFloat(key string) (v float64) {
	switch vv := m[key].(type) {
	case int:
		return float64(vv)
	case string:
		v, _ = strconv.ParseFloat(vv, 64)
		return
	}
	return
}

func (m MapMetadata) GetDuration(key string) (v time.Duration) {
	if m != nil {
		switch vv := m[key].(type) {
		case int:
			return time.Duration(vv) * time.Second
		case string:
			v, _ = time.ParseDuration(vv)
		}
	}
	return
}

func (m MapMetadata) GetString(key string) (v string) {
	if m != nil {
		v, _ = m[key].(string)
	}
	return
}

func (m MapMetadata) GetStrings(key string) (ss []string) {
	if v, _ := m.Get(key).([]interface{}); len(v) > 0 {
		for _, vv := range v {
			if s, ok := vv.(string); ok {
				ss = append(ss, s)
			}
		}
	}
	return
}

func GetStringMapString(md Metadata, key string) (m map[string]string) {
	if mm, _ := md.Get(key).(map[interface{}]interface{}); len(mm) > 0 {
		m = make(map[string]string)
		for k, v := range mm {
			m[fmt.Sprintf("%v", k)] = fmt.Sprintf("%v", v)
		}
	}
	return
}
