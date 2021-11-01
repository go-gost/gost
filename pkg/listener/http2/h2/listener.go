package h2

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/go-gost/gost/pkg/internal/utils"
	"github.com/go-gost/gost/pkg/listener"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"golang.org/x/net/http2"
)

func init() {
	registry.RegisterListener("h2", NewListener)
}

type h2Listener struct {
	addr string
	net.Listener
	md       metadata
	server   *http2.Server
	connChan chan net.Conn
	errChan  chan error
	logger   logger.Logger
}

func NewListener(opts ...listener.Option) listener.Listener {
	options := &listener.Options{}
	for _, opt := range opts {
		opt(options)
	}
	return &h2Listener{
		addr:   options.Addr,
		logger: options.Logger,
	}
}

func (l *h2Listener) Init(md md.Metadata) (err error) {
	if err = l.parseMetadata(md); err != nil {
		return
	}

	ln, err := net.Listen("tcp", l.addr)
	if err != nil {
		return
	}
	l.Listener = &utils.TCPKeepAliveListener{
		TCPListener:     ln.(*net.TCPListener),
		KeepAlivePeriod: l.md.keepAlivePeriod,
	}
	// TODO: tune http2 server config
	l.server = &http2.Server{
		// MaxConcurrentStreams:         1000,
		PermitProhibitedCipherSuites: true,
		IdleTimeout:                  5 * time.Minute,
	}

	queueSize := l.md.connQueueSize
	if queueSize <= 0 {
		queueSize = defaultQueueSize
	}
	l.connChan = make(chan net.Conn, queueSize)
	l.errChan = make(chan error, 1)

	go l.listenLoop()
	return
}

func (l *h2Listener) Accept() (conn net.Conn, err error) {
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

func (l *h2Listener) listenLoop() {
	for {
		conn, err := l.Listener.Accept()
		if err != nil {
			// log.Log("[http2] accept:", err)
			l.errChan <- err
			close(l.errChan)
			return
		}
		go l.handleLoop(conn)
	}
}

func (l *h2Listener) handleLoop(conn net.Conn) {
	if l.md.tlsConfig != nil {
		tlsConn := tls.Server(conn, l.md.tlsConfig)
		// NOTE: HTTP2 server will check the TLS version,
		// so we must ensure that the TLS connection is handshake completed.
		if err := tlsConn.Handshake(); err != nil {
			// log.Logf("[http2] %s - %s : %s", conn.RemoteAddr(), conn.LocalAddr(), err)
			return
		}
		conn = tlsConn
	}

	opt := http2.ServeConnOpts{
		Handler: http.HandlerFunc(l.handleFunc),
	}
	l.server.ServeConn(conn, &opt)
}

func (l *h2Listener) handleFunc(w http.ResponseWriter, r *http.Request) {
	/*
		log.Logf("[http2] %s -> %s %s %s %s",
			r.RemoteAddr, r.Host, r.Method, r.RequestURI, r.Proto)
		if Debug {
			dump, _ := httputil.DumpRequest(r, false)
			log.Log("[http2]", string(dump))
		}
	*/
	// w.Header().Set("Proxy-Agent", "gost/"+Version)
	conn, err := l.upgrade(w, r)
	if err != nil {
		// log.Logf("[http2] %s - %s %s %s %s: %s",
		//	r.RemoteAddr, r.Host, r.Method, r.RequestURI, r.Proto, err)
		return
	}
	select {
	case l.connChan <- conn:
	default:
		conn.Close()
		// log.Logf("[http2] %s - %s: connection queue is full", conn.RemoteAddr(), conn.LocalAddr())
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
		localAddr:  l.Listener.Addr(),
		remoteAddr: remoteAddr,
		closed:     make(chan struct{}),
	}, nil
}

func (l *h2Listener) parseMetadata(md md.Metadata) (err error) {
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
