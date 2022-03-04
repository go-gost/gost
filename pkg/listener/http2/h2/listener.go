package h2

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/http/httputil"

	"github.com/go-gost/gost/pkg/common/metrics"
	"github.com/go-gost/gost/pkg/listener"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func init() {
	registry.ListenerRegistry().Register("h2c", NewListener)
	registry.ListenerRegistry().Register("h2", NewTLSListener)
}

type h2Listener struct {
	server  *http.Server
	addr    net.Addr
	cqueue  chan net.Conn
	errChan chan error
	logger  logger.Logger
	md      metadata
	h2c     bool
	options listener.Options
}

func NewListener(opts ...listener.Option) listener.Listener {
	options := listener.Options{}
	for _, opt := range opts {
		opt(&options)
	}
	return &h2Listener{
		h2c:     true,
		logger:  options.Logger,
		options: options,
	}
}

func NewTLSListener(opts ...listener.Option) listener.Listener {
	options := listener.Options{}
	for _, opt := range opts {
		opt(&options)
	}
	return &h2Listener{
		logger:  options.Logger,
		options: options,
	}
}

func (l *h2Listener) Init(md md.Metadata) (err error) {
	if err = l.parseMetadata(md); err != nil {
		return
	}

	l.server = &http.Server{
		Addr: l.options.Addr,
	}

	ln, err := net.Listen("tcp", l.options.Addr)
	if err != nil {
		return err
	}
	l.addr = ln.Addr()
	ln = metrics.WrapListener(l.options.Service, ln)

	if l.h2c {
		l.server.Handler = h2c.NewHandler(
			http.HandlerFunc(l.handleFunc), &http2.Server{})
	} else {
		l.server.Handler = http.HandlerFunc(l.handleFunc)
		l.server.TLSConfig = l.options.TLSConfig
		if err := http2.ConfigureServer(l.server, nil); err != nil {
			ln.Close()
			return err
		}
		ln = tls.NewListener(ln, l.options.TLSConfig)
	}

	l.cqueue = make(chan net.Conn, l.md.backlog)
	l.errChan = make(chan error, 1)

	go func() {
		if err := l.server.Serve(ln); err != nil {
			l.logger.Error(err)
		}
	}()

	return
}

func (l *h2Listener) Accept() (conn net.Conn, err error) {
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

func (l *h2Listener) Addr() net.Addr {
	return l.addr
}

func (l *h2Listener) Close() (err error) {
	select {
	case <-l.errChan:
	default:
		err = l.server.Close()
		l.errChan <- err
		close(l.errChan)
	}
	return nil
}

func (l *h2Listener) handleFunc(w http.ResponseWriter, r *http.Request) {
	if l.logger.IsLevelEnabled(logger.DebugLevel) {
		dump, _ := httputil.DumpRequest(r, false)
		l.logger.Debug(string(dump))
	}
	conn, err := l.upgrade(w, r)
	if err != nil {
		l.logger.Error(err)
		return
	}
	select {
	case l.cqueue <- conn:
	default:
		conn.Close()
		l.logger.Warnf("connection queue is full, client %s discarded", r.RemoteAddr)
	}

	<-conn.closed // NOTE: we need to wait for streaming end, or the connection will be closed
}

func (l *h2Listener) upgrade(w http.ResponseWriter, r *http.Request) (*conn, error) {
	if l.md.path == "" && r.Method != http.MethodConnect {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return nil, errors.New("method not allowed")
	}

	if l.md.path != "" && r.RequestURI != l.md.path {
		w.WriteHeader(http.StatusBadRequest)
		return nil, errors.New("bad request")
	}

	w.WriteHeader(http.StatusOK)
	if fw, ok := w.(http.Flusher); ok {
		fw.Flush() // write header to client
	}

	remoteAddr, _ := net.ResolveTCPAddr("tcp", r.RemoteAddr)
	if remoteAddr == nil {
		remoteAddr = &net.TCPAddr{
			IP:   net.IPv4zero,
			Port: 0,
		}
	}
	return &conn{
		r:          r.Body,
		w:          flushWriter{w},
		localAddr:  l.addr,
		remoteAddr: remoteAddr,
		closed:     make(chan struct{}),
	}, nil
}
