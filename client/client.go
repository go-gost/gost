package client

import (
	"github.com/go-gost/gost/client/connector"
	"github.com/go-gost/gost/client/transporter"
)

type Client struct {
	Connector   connector.Connector
	Transporter transporter.Transporter
}
