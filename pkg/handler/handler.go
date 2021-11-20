package handler

import (
	"context"
	"net"

	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/metadata"
)

type Handler interface {
	Init(metadata.Metadata) error
	Handle(context.Context, net.Conn)
}

type Forwarder interface {
	Forward(*chain.NodeGroup)
}
