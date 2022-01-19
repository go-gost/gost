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
	case []interface{}:
		for _, vv := range v {
			if s, ok := vv.(string); ok {
				ss = append(ss, s)
			}
		}
	}
	return
}

func GetStringMap(md Metadata, key string) (m map[string]interface{}) {
	switch vv := md.Get(key).(type) {
	case map[string]interface{}:
		return vv
	case map[interface{}]interface{}:
		m = make(map[string]interface{})
		for k, v := range vv {
			m[fmt.Sprintf("%v", k)] = v
		}
	}
	return
}

func GetStringMapString(md Metadata, key string) (m map[string]string) {
	switch vv := md.Get(key).(type) {
	case map[string]interface{}:
		m = make(map[string]string)
		for k, v := range vv {
			m[k] = fmt.Sprintf("%v", v)
		}
	case map[interface{}]interface{}:
		m = make(map[string]string)
		for k, v := range vv {
			m[fmt.Sprintf("%v", k)] = fmt.Sprintf("%v", v)
		}
	}
	return
}
