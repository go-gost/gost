// plain http tunnel

package pht

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-gost/gost/pkg/common/bufpool"
	"github.com/go-gost/gost/pkg/listener"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/rs/xid"
)

func init() {
	registry.RegisterListener("pht", NewListener)
	registry.RegisterListener("phts", NewTLSListener)
}

type phtListener struct {
	tlsEnabled bool
	server     *http.Server
	addr       net.Addr
	conns      sync.Map
	cqueue     chan net.Conn
	errChan    chan error
	logger     logger.Logger
	md         metadata
	options    listener.Options
}

func NewListener(opts ...listener.Option) listener.Listener {
	options := listener.Options{}
	for _, opt := range opts {
		opt(&options)
	}
	return &phtListener{
		logger:  options.Logger,
		options: options,
	}
}

func NewTLSListener(opts ...listener.Option) listener.Listener {
	options := listener.Options{}
	for _, opt := range opts {
		opt(&options)
	}
	return &phtListener{
		tlsEnabled: true,
		logger:     options.Logger,
		options:    options,
	}
}

func (l *phtListener) Init(md md.Metadata) (err error) {
	if err = l.parseMetadata(md); err != nil {
		return
	}

	ln, err := net.Listen("tcp", l.options.Addr)
	if err != nil {
		return err
	}
	l.addr = ln.Addr()

	mux := http.NewServeMux()
	mux.HandleFunc("/authorize", l.handleAuthorize)
	mux.HandleFunc("/push", l.handlePush)
	mux.HandleFunc("/pull", l.handlePull)

	l.server = &http.Server{
		Addr:    l.options.Addr,
		Handler: mux,
	}
	if l.tlsEnabled {
		l.server.TLSConfig = l.options.TLSConfig
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

func (l *phtListener) Accept() (conn net.Conn, err error) {
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

func (l *phtListener) Addr() net.Addr {
	return l.addr
}

func (l *phtListener) Close() (err error) {
	select {
	case <-l.errChan:
	default:
		err = l.server.Close()
		l.errChan <- err
		close(l.errChan)
	}
	return nil
}

func (l *phtListener) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	if l.logger.IsLevelEnabled(logger.DebugLevel) {
		dump, _ := httputil.DumpRequest(r, false)
		l.logger.Debug(string(dump))
	}

	raddr, _ := net.ResolveTCPAddr("tcp", r.RemoteAddr)
	if raddr == nil {
		raddr = &net.TCPAddr{}
	}

	// connection id
	cid := xid.New().String()

	c1, c2 := net.Pipe()
	c := &conn{
		Conn:       c1,
		localAddr:  l.addr,
		remoteAddr: raddr,
	}

	select {
	case l.cqueue <- c:
	default:
		c.Close()
		l.logger.Warnf("connection queue is full, client %s discarded", r.RemoteAddr)
		w.WriteHeader(http.StatusTooManyRequests)
		return
	}

	w.Write([]byte(fmt.Sprintf("token=%s", cid)))
	l.conns.Store(cid, c2)
}

func (l *phtListener) handlePush(w http.ResponseWriter, r *http.Request) {
	if l.logger.IsLevelEnabled(logger.DebugLevel) {
		dump, _ := httputil.DumpRequest(r, false)
		l.logger.Debug(string(dump))
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	cid := r.Form.Get("token")
	v, ok := l.conns.Load(cid)
	if !ok {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	conn := v.(net.Conn)

	br := bufio.NewReader(r.Body)
	data, err := br.ReadString('\n')
	if err != nil {
		l.logger.Error(err)
		conn.Close()
		l.conns.Delete(cid)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	data = strings.TrimSuffix(data, "\n")
	if len(data) == 0 {
		return
	}

	b, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		l.logger.Error(err)
		l.conns.Delete(cid)
		conn.Close()
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
	defer conn.SetWriteDeadline(time.Time{})

	if _, err := conn.Write(b); err != nil {
		l.logger.Error(err)
		l.conns.Delete(cid)
		conn.Close()
		w.WriteHeader(http.StatusGone)
	}
}

func (l *phtListener) handlePull(w http.ResponseWriter, r *http.Request) {
	if l.logger.IsLevelEnabled(logger.DebugLevel) {
		dump, _ := httputil.DumpRequest(r, false)
		l.logger.Debug(string(dump))
	}

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	cid := r.Form.Get("token")
	v, ok := l.conns.Load(cid)
	if !ok {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	conn := v.(net.Conn)

	w.WriteHeader(http.StatusOK)
	if fw, ok := w.(http.Flusher); ok {
		fw.Flush()
	}

	b := bufpool.Get(4096)
	defer bufpool.Put(b)

	for {
		conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		n, err := conn.Read(*b)
		if err != nil {
			if !errors.Is(err, os.ErrDeadlineExceeded) {
				l.logger.Error(err)
				l.conns.Delete(cid)
				conn.Close()
			} else {
				(*b)[0] = '\n'
				w.Write((*b)[:1])
			}
			return
		}

		bw := bufio.NewWriter(w)
		bw.WriteString(base64.StdEncoding.EncodeToString((*b)[:n]))
		bw.WriteString("\n")
		if err := bw.Flush(); err != nil {
			return
		}
		if fw, ok := w.(http.Flusher); ok {
			fw.Flush()
		}
	}
}
