package metadata

import (
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
	GetString(key string) string
	GetDuration(key string) time.Duration
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

func (m MapMetadata) GetString(key string) (v string) {
	if m != nil {
		v, _ = m[key].(string)
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
