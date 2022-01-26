package relay

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/go-gost/gost/pkg/common/bufpool"
	"github.com/go-gost/gost/pkg/common/util/mux"
	"github.com/go-gost/gost/pkg/common/util/socks"
	"github.com/go-gost/gost/pkg/handler"
	"github.com/go-gost/gost/pkg/logger"
	"github.com/go-gost/relay"
)

func (h *relayHandler) handleBind(ctx context.Context, conn net.Conn, network, address string, log logger.Logger) {
	log = log.WithFields(map[string]interface{}{
		"dst": fmt.Sprintf("%s/%s", address, network),
		"cmd": "bind",
	})

	log.Infof("%s >> %s", conn.RemoteAddr(), address)

	resp := relay.Response{
		Version: relay.Version1,
		Status:  relay.StatusOK,
	}

	if !h.md.enableBind {
		resp.Status = relay.StatusForbidden
		resp.WriteTo(conn)
		log.Error("BIND is diabled")
		return
	}

	if network == "tcp" {
		h.bindTCP(ctx, conn, network, address, log)
	} else {
		h.bindUDP(ctx, conn, network, address, log)
	}
}

func (h *relayHandler) bindTCP(ctx context.Context, conn net.Conn, network, address string, log logger.Logger) {
	resp := relay.Response{
		Version: relay.Version1,
		Status:  relay.StatusOK,
	}

	ln, err := net.Listen(network, address) // strict mode: if the port already in use, it will return error
	if err != nil {
		log.Error(err)
		resp.Status = relay.StatusServiceUnavailable
		resp.WriteTo(conn)
		return
	}

	af := &relay.AddrFeature{}
	err = af.ParseFrom(ln.Addr().String())
	if err != nil {
		log.Warn(err)
	}

	// Issue: may not reachable when host has multi-interface
	af.Host, _, _ = net.SplitHostPort(conn.LocalAddr().String())
	af.AType = relay.AddrIPv4
	resp.Features = append(resp.Features, af)
	if _, err := resp.WriteTo(conn); err != nil {
		log.Error(err)
		ln.Close()
		return
	}

	log = log.WithFields(map[string]interface{}{
		"bind": fmt.Sprintf("%s/%s", ln.Addr(), ln.Addr().Network()),
	})
	log.Debugf("bind on %s OK", ln.Addr())

	h.serveTCPBind(ctx, conn, ln, log)
}

func (h *relayHandler) bindUDP(ctx context.Context, conn net.Conn, network, address string, log logger.Logger) {
	resp := relay.Response{
		Version: relay.Version1,
		Status:  relay.StatusOK,
	}

	bindAddr, _ := net.ResolveUDPAddr(network, address)
	pc, err := net.ListenUDP(network, bindAddr)
	if err != nil {
		log.Error(err)
		return
	}
	defer pc.Close()

	af := &relay.AddrFeature{}
	err = af.ParseFrom(pc.LocalAddr().String())
	if err != nil {
		log.Warn(err)
	}

	// Issue: may not reachable when host has multi-interface
	af.Host, _, _ = net.SplitHostPort(conn.LocalAddr().String())
	af.AType = relay.AddrIPv4
	resp.Features = append(resp.Features, af)
	if _, err := resp.WriteTo(conn); err != nil {
		log.Error(err)
		return
	}

	log = log.WithFields(map[string]interface{}{
		"bind": pc.LocalAddr().String(),
	})
	log.Debugf("bind on %s OK", pc.LocalAddr())

	t := time.Now()
	log.Infof("%s <-> %s", conn.RemoteAddr(), pc.LocalAddr())
	h.tunnelServerUDP(
		socks.UDPTunServerConn(conn),
		pc,
		log,
	)
	log.WithFields(map[string]interface{}{
		"duration": time.Since(t),
	}).Infof("%s >-< %s", conn.RemoteAddr(), pc.LocalAddr())
}

func (h *relayHandler) serveTCPBind(ctx context.Context, conn net.Conn, ln net.Listener, log logger.Logger) {
	// Upgrade connection to multiplex stream.
	session, err := mux.ClientSession(conn)
	if err != nil {
		log.Error(err)
		return
	}
	defer session.Close()

	go func() {
		defer ln.Close()
		for {
			conn, err := session.Accept()
			if err != nil {
				log.Error(err)
				return
			}
			conn.Close() // we do not handle incoming connections.
		}
	}()

	for {
		rc, err := ln.Accept()
		if err != nil {
			log.Error(err)
			return
		}
		log.Debugf("peer %s accepted", rc.RemoteAddr())

		go func(c net.Conn) {
			defer c.Close()

			log = log.WithFields(map[string]interface{}{
				"local":  ln.Addr().String(),
				"remote": c.RemoteAddr().String(),
			})

			sc, err := session.GetConn()
			if err != nil {
				log.Error(err)
				return
			}
			defer sc.Close()

			af := &relay.AddrFeature{}
			af.ParseFrom(c.RemoteAddr().String())
			resp := relay.Response{
				Version:  relay.Version1,
				Status:   relay.StatusOK,
				Features: []relay.Feature{af},
			}
			if _, err := resp.WriteTo(sc); err != nil {
				log.Error(err)
				return
			}

			t := time.Now()
			log.Infof("%s <-> %s", c.LocalAddr(), c.RemoteAddr())
			handler.Transport(sc, c)
			log.WithFields(map[string]interface{}{"duration": time.Since(t)}).
				Infof("%s >-< %s", c.LocalAddr(), c.RemoteAddr())
		}(rc)
	}
}

func (h *relayHandler) tunnelServerUDP(tunnel, c net.PacketConn, log logger.Logger) (err error) {
	bufSize := h.md.udpBufferSize
	errc := make(chan error, 2)

	go func() {
		for {
			err := func() error {
				b := bufpool.Get(bufSize)
				defer bufpool.Put(b)

				n, raddr, err := tunnel.ReadFrom(*b)
				if err != nil {
					return err
				}

				if h.options.Bypass != nil && h.options.Bypass.Contains(raddr.String()) {
					log.Warn("bypass: ", raddr)
					return nil
				}

				if _, err := c.WriteTo((*b)[:n], raddr); err != nil {
					return err
				}

				log.Debugf("%s >>> %s data: %d",
					c.LocalAddr(), raddr, n)

				return nil
			}()

			if err != nil {
				errc <- err
				return
			}
		}
	}()

	go func() {
		for {
			err := func() error {
				b := bufpool.Get(bufSize)
				defer bufpool.Put(b)

				n, raddr, err := c.ReadFrom(*b)
				if err != nil {
					return err
				}

				if h.options.Bypass != nil && h.options.Bypass.Contains(raddr.String()) {
					log.Warn("bypass: ", raddr)
					return nil
				}

				if _, err := tunnel.WriteTo((*b)[:n], raddr); err != nil {
					return err
				}
				log.Debugf("%s <<< %s data: %d",
					c.LocalAddr(), raddr, n)

				return nil
			}()

			if err != nil {
				errc <- err
				return
			}
		}
	}()

	return <-errc
}
