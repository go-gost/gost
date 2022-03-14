package handler

import (
	"context"
	"net"

	"github.com/go-gost/gost/v3/pkg/chain"
	"github.com/go-gost/gost/v3/pkg/metadata"
)

type Handler interface {
	Init(metadata.Metadata) error
	Handle(context.Context, net.Conn, ...HandleOption) error
}

type Forwarder interface {
	Forward(*chain.NodeGroup)
}
