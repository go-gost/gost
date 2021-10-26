package handler

import (
	"context"
	"net"
)

type Handler interface {
	Handle(context.Context, net.Conn)
}
