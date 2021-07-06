package http2

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"

	"github.com/go-gost/gost/logger"
	"github.com/go-gost/gost/server/listener"
	"github.com/go-gost/gost/utils"
	"golang.org/x/net/http2"
)

var (
	_ listener.Listener = (*Listener)(nil)
)

type Listener struct {
	md       metadata
	server   *http.Server
	addr     net.Addr
	connChan chan *conn
	errChan  chan error
	logger   logger.Logger
}

func NewListener(opts ...listener.Option) *Listener {
	options := &listener.Options{}
	for _, opt := range opts {
		opt(options)
	}
	return &Listener{
		logger: options.Logger,
	}
}

func (l *Listener) Init(md listener.Metadata) (err error) {
	l.md, err = l.parseMetadata(md)
	if err != nil {
		return
	}

	l.server = &http.Server{
		Addr:      l.md.addr,
		Handler:   http.HandlerFunc(l.handleFunc),
		TLSConfig: l.md.tlsConfig,
	}
	if err := http2.ConfigureServer(l.server, nil); err != nil {
		return err
	}

	ln, err := net.Listen("tcp", addr)
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

func (l *Listener) parseMetadata(md listener.Metadata) (m metadata, err error) {
	if val, ok := md[addr]; ok {
		m.addr = val
	} else {
		err = errors.New("missing address")
		return
	}

	m.tlsConfig, err = utils.LoadTLSConfig(md[certFile], md[keyFile], md[caFile])
	if err != nil {
		return
	}

	return
}
