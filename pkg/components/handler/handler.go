package handler

import (
	"context"
	"net"

	"github.com/go-gost/gost/pkg/components/metadata"
)

type Handler interface {
	Init(metadata.Metadata) error
	Handle(context.Context, net.Conn)
}
