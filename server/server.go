package server

import (
	"github.com/go-gost/gost/server/handler"
	"github.com/go-gost/gost/server/listener"
)

// Server is a proxy server.
type Server struct {
	Handler  handler.Handler
	Listener listener.Listener
}
