package metadata

import (
	"fmt"
	"strconv"
	"time"
)

type Metadatable interface {
	GetMetadata() Metadata
}

type Metadata interface {
	IsExists(key string) bool
	Set(key string, value any)
	Get(key string) any
}

type MapMetadata map[string]any

func (m MapMetadata) IsExists(key string) bool {
	_, ok := m[key]
	return ok
}

func (m MapMetadata) Set(key string, value any) {
	m[key] = value
}

func (m MapMetadata) Get(key string) any {
	if m != nil {
		return m[key]
	}
	return nil
}

func (m MapMetadata) Del(key string) {
	delete(m, key)
}

func GetBool(md Metadata, key string) (v bool) {
	if md == nil || !md.IsExists(key) {
		return
	}
	switch vv := md.Get(key).(type) {
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

func GetInt(md Metadata, key string) (v int) {
	if md == nil {
		return
	}

	switch vv := md.Get(key).(type) {
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

func GetFloat(md Metadata, key string) (v float64) {
	if md == nil {
		return
	}

	switch vv := md.Get(key).(type) {
	case int:
		return float64(vv)
	case string:
		v, _ = strconv.ParseFloat(vv, 64)
		return
	}
	return
}

func GetDuration(md Metadata, key string) (v time.Duration) {
	if md == nil {
		return
	}
	switch vv := md.Get(key).(type) {
	case int:
		return time.Duration(vv) * time.Second
	case string:
		v, _ = time.ParseDuration(vv)
		if v == 0 {
			n, _ := strconv.Atoi(vv)
			v = time.Duration(n) * time.Second
		}
	}
	return
}

func GetString(md Metadata, key string) (v string) {
	if md != nil {
		v, _ = md.Get(key).(string)
	}
	return
}

func GetStrings(md Metadata, key string) (ss []string) {
	switch v := md.Get(key).(type) {
	case []string:
		ss = v
	case []any:
		for _, vv := range v {
			if s, ok := vv.(string); ok {
				ss = append(ss, s)
			}
		}
	}
	return
}

func GetStringMap(md Metadata, key string) (m map[string]any) {
	switch vv := md.Get(key).(type) {
	case map[string]any:
		return vv
	case map[any]any:
		m = make(map[string]any)
		for k, v := range vv {
			m[fmt.Sprintf("%v", k)] = v
		}
	}
	return
}

func GetStringMapString(md Metadata, key string) (m map[string]string) {
	switch vv := md.Get(key).(type) {
	case map[string]any:
		m = make(map[string]string)
		for k, v := range vv {
			m[k] = fmt.Sprintf("%v", v)
		}
	case map[any]any:
		m = make(map[string]string)
		for k, v := range vv {
			m[fmt.Sprintf("%v", k)] = fmt.Sprintf("%v", v)
		}
	}
	return
}
