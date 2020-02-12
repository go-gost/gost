package gost

import (
	"fmt"
	"sync"
)

var (
	mConnectors sync.Map
)

type ConnectorCreator func(params Params) (Connector, error)

func RegisterConnector(name string, creator ConnectorCreator) {
	mConnectors.Store(name, creator)
}

func NewConnector(name string, params Params) (Connector, error) {
	v, ok := mConnectors.Load(name)
	if !ok || v == nil {
		return nil, fmt.Errorf("connector %s not found", name)
	}
	return (v.(ConnectorCreator))(params)
}
