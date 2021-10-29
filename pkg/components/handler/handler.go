package handler

import (
	"context"
	"net"
)

type Handler interface {
	Init(Metadata) error
	Handle(context.Context, net.Conn)
}
