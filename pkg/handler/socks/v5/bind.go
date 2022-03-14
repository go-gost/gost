package v5

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/go-gost/gosocks5"
	netpkg "github.com/go-gost/gost/v3/pkg/common/net"
	"github.com/go-gost/gost/v3/pkg/logger"
)

func (h *socks5Handler) handleBind(ctx context.Context, conn net.Conn, network, address string, log logger.Logger) error {
	log = log.WithFields(map[string]any{
		"dst": fmt.Sprintf("%s/%s", address, network),
		"cmd": "bind",
	})

	log.Infof("%s >> %s", conn.RemoteAddr(), address)

	if !h.md.enableBind {
		reply := gosocks5.NewReply(gosocks5.NotAllowed, nil)
		log.Debug(reply)
		log.Error("socks5: BIND is disabled")
		return reply.Write(conn)
	}

	// BIND does not support chain.
	return h.bindLocal(ctx, conn, network, address, log)
}

func (h *socks5Handler) bindLocal(ctx context.Context, conn net.Conn, network, address string, log logger.Logger) error {
	ln, err := net.Listen(network, address) // strict mode: if the port already in use, it will return error
	if err != nil {
		log.Error(err)
		reply := gosocks5.NewReply(gosocks5.Failure, nil)
		if err := reply.Write(conn); err != nil {
			log.Error(err)
		}
		log.Debug(reply)
		return err
	}

	socksAddr := gosocks5.Addr{}
	if err := socksAddr.ParseFrom(ln.Addr().String()); err != nil {
		log.Warn(err)
	}

	// Issue: may not reachable when host has multi-interface
	socksAddr.Host, _, _ = net.SplitHostPort(conn.LocalAddr().String())
	socksAddr.Type = 0
	reply := gosocks5.NewReply(gosocks5.Succeeded, &socksAddr)
	if err := reply.Write(conn); err != nil {
		log.Error(err)
		ln.Close()
		return err
	}
	log.Debug(reply)

	log = log.WithFields(map[string]any{
		"bind": fmt.Sprintf("%s/%s", ln.Addr(), ln.Addr().Network()),
	})

	log.Debugf("bind on %s OK", ln.Addr())

	h.serveBind(ctx, conn, ln, log)
	return nil
}

func (h *socks5Handler) serveBind(ctx context.Context, conn net.Conn, ln net.Listener, log logger.Logger) {
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

			errc <- netpkg.Transport(conn, pc1)
		}()

		return errc
	}

	defer pc2.Close()

	select {
	case err := <-accept():
		if err != nil {
			log.Error(err)

			reply := gosocks5.NewReply(gosocks5.Failure, nil)
			if err := reply.Write(pc2); err != nil {
				log.Error(err)
			}
			log.Debug(reply)

			return
		}
		defer rc.Close()

		log.Debugf("peer %s accepted", rc.RemoteAddr())

		log = log.WithFields(map[string]any{
			"local":  rc.LocalAddr().String(),
			"remote": rc.RemoteAddr().String(),
		})

		raddr := gosocks5.Addr{}
		raddr.ParseFrom(rc.RemoteAddr().String())
		reply := gosocks5.NewReply(gosocks5.Succeeded, &raddr)
		if err := reply.Write(pc2); err != nil {
			log.Error(err)
		}
		log.Debug(reply)

		start := time.Now()
		log.Infof("%s <-> %s", rc.LocalAddr(), rc.RemoteAddr())
		netpkg.Transport(pc2, rc)
		log.WithFields(map[string]any{"duration": time.Since(start)}).
			Infof("%s >-< %s", rc.LocalAddr(), rc.RemoteAddr())

	case err := <-pipe():
		if err != nil {
			log.Error(err)
		}
		ln.Close()
		return
	}
}
