package client

import (
	"github.com/go-gost/gost/client/connector"
	"github.com/go-gost/gost/client/dialer"
)

type Client struct {
	Connector connector.Connector
	Dialer    dialer.Dialer
}
