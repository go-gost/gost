package v5

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/pkg/handler"
)

func (h *socks5Handler) handleBind(ctx context.Context, conn net.Conn, network, address string) {
	h.logger = h.logger.WithFields(map[string]interface{}{
		"dst": fmt.Sprintf("%s/%s", address, network),
		"cmd": "bind",
	})

	h.logger.Infof("%s >> %s", conn.RemoteAddr(), address)

	if !h.md.enableBind {
		reply := gosocks5.NewReply(gosocks5.NotAllowed, nil)
		reply.Write(conn)
		h.logger.Debug(reply)
		h.logger.Error("BIND is diabled")
		return
	}

	// BIND does not support chain.
	h.bindLocal(ctx, conn, network, address)
}

func (h *socks5Handler) bindLocal(ctx context.Context, conn net.Conn, network, address string) {
	ln, err := net.Listen(network, address) // strict mode: if the port already in use, it will return error
	if err != nil {
		h.logger.Error(err)
		reply := gosocks5.NewReply(gosocks5.Failure, nil)
		if err := reply.Write(conn); err != nil {
			h.logger.Error(err)
		}
		h.logger.Debug(reply)
		return
	}

	socksAddr := gosocks5.Addr{}
	if err := socksAddr.ParseFrom(ln.Addr().String()); err != nil {
		h.logger.Warn(err)
	}

	// Issue: may not reachable when host has multi-interface
	socksAddr.Host, _, _ = net.SplitHostPort(conn.LocalAddr().String())
	socksAddr.Type = 0
	reply := gosocks5.NewReply(gosocks5.Succeeded, &socksAddr)
	if err := reply.Write(conn); err != nil {
		h.logger.Error(err)
		ln.Close()
		return
	}
	h.logger.Debug(reply)

	h.logger = h.logger.WithFields(map[string]interface{}{
		"bind": fmt.Sprintf("%s/%s", ln.Addr(), ln.Addr().Network()),
	})

	h.logger.Debugf("bind on %s OK", ln.Addr())

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

			reply := gosocks5.NewReply(gosocks5.Failure, nil)
			if err := reply.Write(pc2); err != nil {
				h.logger.Error(err)
			}
			h.logger.Debug(reply)

			return
		}
		defer rc.Close()

		h.logger.Debugf("peer %s accepted", rc.RemoteAddr())

		raddr := gosocks5.Addr{}
		raddr.ParseFrom(rc.RemoteAddr().String())
		reply := gosocks5.NewReply(gosocks5.Succeeded, &raddr)
		if err := reply.Write(pc2); err != nil {
			h.logger.Error(err)
		}
		h.logger.Debug(reply)

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
