package http2

import (
	"crypto/tls"
	"net"
	"net/http"

	"github.com/go-gost/gost/pkg/common/util"
	http2_util "github.com/go-gost/gost/pkg/internal/util/http2"
	"github.com/go-gost/gost/pkg/listener"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"golang.org/x/net/http2"
)

func init() {
	registry.RegisterListener("http2", NewListener)
}

type http2Listener struct {
	server  *http.Server
	saddr   string
	addr    net.Addr
	cqueue  chan net.Conn
	errChan chan error
	logger  logger.Logger
	md      metadata
}

func NewListener(opts ...listener.Option) listener.Listener {
	options := &listener.Options{}
	for _, opt := range opts {
		opt(options)
	}
	return &http2Listener{
		saddr:  options.Addr,
		logger: options.Logger,
	}
}

func (l *http2Listener) Init(md md.Metadata) (err error) {
	if err = l.parseMetadata(md); err != nil {
		return
	}

	l.server = &http.Server{
		Addr:      l.saddr,
		Handler:   http.HandlerFunc(l.handleFunc),
		TLSConfig: l.md.tlsConfig,
	}
	if err := http2.ConfigureServer(l.server, nil); err != nil {
		return err
	}

	ln, err := net.Listen("tcp", l.saddr)
	if err != nil {
		return err
	}
	l.addr = ln.Addr()

	ln = tls.NewListener(
		&util.TCPKeepAliveListener{
			TCPListener: ln.(*net.TCPListener),
		},
		l.md.tlsConfig,
	)

	l.cqueue = make(chan net.Conn, l.md.backlog)
	l.errChan = make(chan error, 1)

	go func() {
		if err := l.server.Serve(ln); err != nil {
			l.logger.Error(err)
		}
	}()

	return
}

func (l *http2Listener) Accept() (conn net.Conn, err error) {
	var ok bool
	select {
	case conn = <-l.cqueue:
	case err, ok = <-l.errChan:
		if !ok {
			err = listener.ErrClosed
		}
	}
	return
}

func (l *http2Listener) Addr() net.Addr {
	return l.addr
}

func (l *http2Listener) Close() (err error) {
	select {
	case <-l.errChan:
	default:
		err = l.server.Close()
		l.errChan <- err
		close(l.errChan)
	}
	return nil
}

func (l *http2Listener) handleFunc(w http.ResponseWriter, r *http.Request) {
	conn := http2_util.NewServerConn(w, r)
	select {
	case l.cqueue <- conn:
	default:
		l.logger.Warnf("connection queue is full, client %s discarded", r.RemoteAddr)
		return
	}

	<-conn.Done()
}
