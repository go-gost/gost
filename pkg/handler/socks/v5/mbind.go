package v5

import (
	"context"
	"net"
	"time"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/handler"
	"github.com/go-gost/gost/pkg/internal/utils/mux"
)

func (h *socks5Handler) handleMuxBind(ctx context.Context, conn net.Conn, req *gosocks5.Request) {
	addr := req.Addr.String()

	h.logger = h.logger.WithFields(map[string]interface{}{
		"dst": addr,
		"cmd": "mbind",
	})

	h.logger.Infof("%s >> %s", conn.RemoteAddr(), addr)

	if !h.md.enableBind {
		reply := gosocks5.NewReply(gosocks5.NotAllowed, nil)
		reply.Write(conn)
		h.logger.Debug(reply)
		h.logger.Error("BIND is diabled")
		return
	}

	if h.chain.IsEmpty() {
		h.muxBindLocal(ctx, conn, addr)
		return
	}

	r := (&chain.Router{}).
		WithChain(h.chain).
		WithRetry(h.md.retryCount).
		WithLogger(h.logger)
	cc, err := r.Connect(ctx)
	if err != nil {
		resp := gosocks5.NewReply(gosocks5.Failure, nil)
		resp.Write(conn)
		h.logger.Debug(resp)
		return
	}
	defer cc.Close()

	// forward request
	if err := req.Write(cc); err != nil {
		h.logger.Error(err)
		resp := gosocks5.NewReply(gosocks5.NetUnreachable, nil)
		resp.Write(conn)
		h.logger.Debug(resp)
		return
	}

	t := time.Now()
	h.logger.Infof("%s <-> %s", conn.RemoteAddr(), addr)
	handler.Transport(conn, cc)
	h.logger.
		WithFields(map[string]interface{}{
			"duration": time.Since(t),
		}).
		Infof("%s >-< %s", conn.RemoteAddr(), addr)
}

func (h *socks5Handler) muxBindLocal(ctx context.Context, conn net.Conn, addr string) {
	bindAddr, _ := net.ResolveTCPAddr("tcp", addr)
	ln, err := net.ListenTCP("tcp", bindAddr) // strict mode: if the port already in use, it will return error
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
	socksAddr.ParseFrom(ln.Addr().String())
	if err != nil {
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
		"bind": socksAddr.String(),
	})
	h.logger.Debugf("bind on %s OK", &socksAddr)

	h.serveMuxBind(ctx, conn, ln)
}

func (h *socks5Handler) serveMuxBind(ctx context.Context, conn net.Conn, ln net.Listener) {
	// Upgrade connection to multiplex stream.
	session, err := mux.ClientSession(conn)
	if err != nil {
		h.logger.Error(err)
		return
	}
	defer session.Close()

	go func() {
		defer ln.Close()
		for {
			conn, err := session.Accept()
			if err != nil {
				h.logger.Error(err)
				return
			}
			conn.Close() // we do not handle incoming connections.
		}
	}()

	for {
		rc, err := ln.Accept()
		if err != nil {
			h.logger.Error(err)
			return
		}
		h.logger.Debugf("peer %s accepted", rc.RemoteAddr())

		go func(c net.Conn) {
			defer c.Close()

			sc, err := session.GetConn()
			if err != nil {
				h.logger.Error(err)
				return
			}
			defer sc.Close()

			// incompatible with GOST v2.x
			if !h.md.compatibilityMode {
				addr := gosocks5.Addr{}
				addr.ParseFrom(c.RemoteAddr().String())
				reply := gosocks5.NewReply(gosocks5.Succeeded, &addr)
				if err := reply.Write(sc); err != nil {
					h.logger.Error(err)
					return
				}
				h.logger.Debug(reply)
			}

			t := time.Now()
			h.logger.Infof("%s <-> %s", conn.RemoteAddr(), c.RemoteAddr().String())
			handler.Transport(sc, c)
			h.logger.
				WithFields(map[string]interface{}{"duration": time.Since(t)}).
				Infof("%s >-< %s", conn.RemoteAddr(), c.RemoteAddr().String())
		}(rc)
	}
}
