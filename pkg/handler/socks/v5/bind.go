package v5

import (
	"context"
	"net"
	"time"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/pkg/handler"
	"github.com/go-gost/gost/pkg/logger"
)

func (h *socks5Handler) handleBind(ctx context.Context, conn net.Conn, req *gosocks5.Request) {
	addr := req.Addr.String()

	h.logger = h.logger.WithFields(map[string]interface{}{
		"dst": addr,
		"cmd": "bind",
	})

	h.logger.Infof("%s >> %s", conn.RemoteAddr(), addr)

	if h.chain.IsEmpty() {
		h.bindLocal(ctx, conn, addr)
		return
	}

	r := (&handler.Router{}).
		WithChain(h.chain).
		WithRetry(h.md.retryCount).
		WithLogger(h.logger)
	cc, err := r.Connect(ctx)
	if err != nil {
		resp := gosocks5.NewReply(gosocks5.Failure, nil)
		resp.Write(conn)
		if h.logger.IsLevelEnabled(logger.DebugLevel) {
			h.logger.Debug(resp)
		}
		return
	}
	defer cc.Close()

	// forward request
	if err := req.Write(cc); err != nil {
		h.logger.Error(err)
		resp := gosocks5.NewReply(gosocks5.NetUnreachable, nil)
		resp.Write(conn)
		if h.logger.IsLevelEnabled(logger.DebugLevel) {
			h.logger.Debug(resp)
		}
		return
	}

	h.logger.Infof("%s <-> %s", conn.RemoteAddr(), addr)
	handler.Transport(conn, cc)
	h.logger.Infof("%s >-< %s", conn.RemoteAddr(), addr)
}

func (h *socks5Handler) bindLocal(ctx context.Context, conn net.Conn, addr string) {
	bindAddr, _ := net.ResolveTCPAddr("tcp", addr)
	ln, err := net.ListenTCP("tcp", bindAddr) // strict mode: if the port already in use, it will return error
	if err != nil {
		h.logger.Error(err)
		reply := gosocks5.NewReply(gosocks5.Failure, nil)
		if err := reply.Write(conn); err != nil {
			h.logger.Error(err)
		}
		if h.logger.IsLevelEnabled(logger.DebugLevel) {
			h.logger.Debug(reply.String())
		}
		return
	}

	socksAddr, err := gosocks5.NewAddr(ln.Addr().String())
	if err != nil {
		h.logger.Warn(err)
		socksAddr = &gosocks5.Addr{
			Type: gosocks5.AddrIPv4,
		}
	}

	// Issue: may not reachable when host has multi-interface
	socksAddr.Host, _, _ = net.SplitHostPort(conn.LocalAddr().String())
	socksAddr.Type = 0
	reply := gosocks5.NewReply(gosocks5.Succeeded, socksAddr)
	if err := reply.Write(conn); err != nil {
		h.logger.Error(err)
		ln.Close()
		return
	}
	if h.logger.IsLevelEnabled(logger.DebugLevel) {
		h.logger.Debug(reply.String())
	}

	h.logger = h.logger.WithFields(map[string]interface{}{
		"bind": socksAddr.String(),
	})
	h.logger.Infof("bind on %s OK", socksAddr.String())

	h.serveBind(ctx, conn, ln)
}

func (h *socks5Handler) serveBind(ctx context.Context, conn net.Conn, ln net.Listener) {
	var rc net.Conn
	accept := func() <-chan error {
		errc := make(chan error, 1)

		go func() {
			defer close(errc)
			defer ln.Close()

			c, err := ln.Accept()
			if err != nil {
				errc <- err
			}
			rc = c
		}()

		return errc
	}

	pc1, pc2 := net.Pipe()
	pipe := func() <-chan error {
		errc := make(chan error, 1)

		go func() {
			defer close(errc)
			defer pc1.Close()

			errc <- handler.Transport(conn, pc1)
		}()

		return errc
	}

	defer pc2.Close()

	select {
	case err := <-accept():
		if err != nil {
			h.logger.Error(err)
			return
		}
		defer rc.Close()

		raddr, _ := gosocks5.NewAddr(rc.RemoteAddr().String())
		reply := gosocks5.NewReply(gosocks5.Succeeded, raddr)
		if err := reply.Write(pc2); err != nil {
			h.logger.Error(err)
		}
		if h.logger.IsLevelEnabled(logger.DebugLevel) {
			h.logger.Debug(reply.String())
		}
		h.logger.Infof("peer accepted: %s", raddr.String())

		start := time.Now()
		h.logger.Infof("%s <-> %s", conn.RemoteAddr(), raddr.String())
		handler.Transport(pc2, rc)
		h.logger.
			WithFields(map[string]interface{}{"duration": time.Since(start)}).
			Infof("%s >-< %s", conn.RemoteAddr(), raddr.String())

	case err := <-pipe():
		if err != nil {
			h.logger.Error(err)
		}
		ln.Close()
		return
	}
}
