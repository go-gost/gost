package http2

import (
	"crypto/tls"
	"net"
	"net/http"

	"github.com/go-gost/gost/pkg/components/internal/utils"
	"github.com/go-gost/gost/pkg/components/listener"
	md "github.com/go-gost/gost/pkg/components/metadata"
	"github.com/go-gost/gost/pkg/logger"
	"github.com/go-gost/gost/pkg/registry"
	"golang.org/x/net/http2"
)

func init() {
	registry.RegisterListener("http2", NewListener)
}

type Listener struct {
	saddr    string
	md       metadata
	server   *http.Server
	addr     net.Addr
	connChan chan *conn
	errChan  chan error
	logger   logger.Logger
}

func NewListener(opts ...listener.Option) listener.Listener {
	options := &listener.Options{}
	for _, opt := range opts {
		opt(options)
	}
	return &Listener{
		saddr:  options.Addr,
		logger: options.Logger,
	}
}

func (l *Listener) Init(md md.Metadata) (err error) {
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
		&utils.TCPKeepAliveListener{
			TCPListener:     ln.(*net.TCPListener),
			KeepAlivePeriod: l.md.keepAlivePeriod,
		},
		l.md.tlsConfig,
	)

	queueSize := l.md.connQueueSize
	if queueSize <= 0 {
		queueSize = defaultQueueSize
	}
	l.connChan = make(chan *conn, queueSize)
	l.errChan = make(chan error, 1)

	go func() {
		if err := l.server.Serve(ln); err != nil {
			// log.Log("[http2]", err)
		}
	}()

	return
}

func (l *Listener) Accept() (conn net.Conn, err error) {
	var ok bool
	select {
	case conn = <-l.connChan:
	case err, ok = <-l.errChan:
		if !ok {
			err = listener.ErrClosed
		}
	}
	return
}

func (l *Listener) Addr() net.Addr {
	return l.addr
}

func (l *Listener) Close() (err error) {
	select {
	case <-l.errChan:
	default:
		err = l.server.Close()
		l.errChan <- err
		close(l.errChan)
	}
	return nil
}

func (l *Listener) handleFunc(w http.ResponseWriter, r *http.Request) {
	conn := &conn{
		r:      r,
		w:      w,
		closed: make(chan struct{}),
	}
	select {
	case l.connChan <- conn:
	default:
		// log.Logf("[http2] %s - %s: connection queue is full", r.RemoteAddr, l.server.Addr)
		return
	}

	<-conn.closed
}

func (l *Listener) parseMetadata(md md.Metadata) (err error) {
	l.md.tlsConfig, err = utils.LoadTLSConfig(
		md.GetString(certFile),
		md.GetString(keyFile),
		md.GetString(caFile),
	)
	if err != nil {
		return
	}

	return
}
