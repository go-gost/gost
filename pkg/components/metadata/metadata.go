package metadata

import "time"

type Metadata interface {
	Get(key string) interface{}
	GetBool(key string) bool
	GetInt(key string) int
	GetFloat(key string) float64
	GetString(key string) string
	GetDuration(key string) time.Duration
}

type MapMetadata map[string]interface{}

func (m MapMetadata) Get(key string) interface{} {
	if m != nil {
		return m[key]
	}
	return nil
}

func (m MapMetadata) GetBool(key string) (v bool) {
	if m != nil {
		v, _ = m[key].(bool)
	}
	return
}

func (m MapMetadata) GetInt(key string) (v int) {
	if m != nil {
		v, _ = m[key].(int)
	}
	return
}

func (m MapMetadata) GetFloat(key string) (v float64) {
	if m != nil {
		v, _ = m[key].(float64)
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
