package gost

import (
	"fmt"
	"sync"
)

var (
	mDialers sync.Map
)

type DialerCreator func(params Params) (Dialer, error)

func RegisterDialer(name string, creator DialerCreator) {
	mDialers.Store(name, creator)
}

func NewDialer(name string, params Params) (Dialer, error) {
	v, ok := mDialers.Load(name)
	if !ok || v == nil {
		return nil, fmt.Errorf("dialer %s not found", name)
	}
	return (v.(DialerCreator))(params)
}
