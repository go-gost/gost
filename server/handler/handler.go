package handler

import (
	"context"
	"net"
)

type Handler interface {
	Init(md Metadata) error
	Handle(context.Context, net.Conn)
}
