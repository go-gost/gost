package auto

import (
	"bufio"
	"context"
	"net"

	"github.com/go-gost/gosocks4"
	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/pkg/handler"
	http_handler "github.com/go-gost/gost/pkg/handler/http"
	socks4_handler "github.com/go-gost/gost/pkg/handler/socks/v4"
	socks5_handler "github.com/go-gost/gost/pkg/handler/socks/v5"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.RegisterHandler("auto", NewHandler)
}

type autoHandler struct {
	httpHandler   handler.Handler
	socks4Handler handler.Handler
	socks5Handler handler.Handler
	log           logger.Logger
}

func NewHandler(opts ...handler.Option) handler.Handler {
	options := &handler.Options{}
	for _, opt := range opts {
		opt(options)
	}

	log := options.Logger
	if log == nil {
		log = logger.Default()
	}

	h := &autoHandler{
		log: log,
	}

	v := append(opts,
		handler.LoggerOption(log.WithFields(map[string]interface{}{"type": "http"})))
	h.httpHandler = http_handler.NewHandler(v...)

	v = append(opts,
		handler.LoggerOption(log.WithFields(map[string]interface{}{"type": "socks4"})))
	h.socks4Handler = socks4_handler.NewHandler(v...)

	v = append(opts,
		handler.LoggerOption(log.WithFields(map[string]interface{}{"type": "socks5"})))
	h.socks5Handler = socks5_handler.NewHandler(v...)
	return h
}

func (h *autoHandler) Init(md md.Metadata) error {
	if err := h.httpHandler.Init(md); err != nil {
		return err
	}
	if err := h.socks4Handler.Init(md); err != nil {
		return err
	}
	if err := h.socks5Handler.Init(md); err != nil {
		return err
	}
	return nil
}

func (h *autoHandler) Handle(ctx context.Context, conn net.Conn) {
	h.log = h.log.WithFields(map[string]interface{}{
		"remote": conn.RemoteAddr().String(),
		"local":  conn.LocalAddr().String(),
	})

	br := bufio.NewReader(conn)
	b, err := br.Peek(1)
	if err != nil {
		h.log.Error(err)
		conn.Close()
		return
	}

	cc := handler.NewBufferReaderConn(conn, br)
	switch b[0] {
	case gosocks4.Ver4: // socks4
		h.socks4Handler.Handle(ctx, cc)
	case gosocks5.Ver5: // socks5
		h.socks5Handler.Handle(ctx, cc)
	default: // http
		h.httpHandler.Handle(ctx, cc)
	}

}
