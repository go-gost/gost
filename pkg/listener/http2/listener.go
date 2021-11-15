package http2

import (
	"crypto/tls"
	"net"
	"net/http"

	"github.com/go-gost/gost/pkg/common/util"
	tls_util "github.com/go-gost/gost/pkg/common/util/tls"
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

func (l *http2Listener) Accept() (conn net.Conn, err error) {
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

func (l *http2Listener) parseMetadata(md md.Metadata) (err error) {
	l.md.tlsConfig, err = tls_util.LoadTLSConfig(
		md.GetString(certFile),
		md.GetString(keyFile),
		md.GetString(caFile),
	)
	if err != nil {
		return
	}

	return
}
