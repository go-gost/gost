package v5

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"time"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/pkg/handler"
	"github.com/go-gost/gost/pkg/internal/bufpool"
	"github.com/go-gost/gost/pkg/internal/utils/socks"
)

func (h *socks5Handler) handleUDP(ctx context.Context, conn net.Conn, req *gosocks5.Request) {
	h.logger = h.logger.WithFields(map[string]interface{}{
		"cmd": "udp",
	})

	relay, err := net.ListenUDP("udp", nil)
	if err != nil {
		h.logger.Error(err)
		reply := gosocks5.NewReply(gosocks5.Failure, nil)
		reply.Write(conn)
		h.logger.Debug(reply)
		return
	}
	defer relay.Close()

	saddr := gosocks5.Addr{}
	saddr.ParseFrom(relay.LocalAddr().String())
	saddr.Type = 0
	saddr.Host, _, _ = net.SplitHostPort(conn.LocalAddr().String()) // replace the IP to the out-going interface's
	reply := gosocks5.NewReply(gosocks5.Succeeded, &saddr)
	if err := reply.Write(conn); err != nil {
		h.logger.Error(err)
		return
	}
	h.logger.Debug(reply)

	h.logger = h.logger.WithFields(map[string]interface{}{
		"bind": saddr.String(),
	})
	h.logger.Infof("bind on %s OK", saddr.String())

	if h.chain.IsEmpty() {
		// serve as standard socks5 udp relay.
		peer, err := net.ListenUDP("udp", nil)
		if err != nil {
			h.logger.Error(err)
			return
		}
		defer peer.Close()

		go h.relayUDP(
			socks.NewUDPConn(relay, h.md.udpBufferSize),
			peer,
		)
	} else {
		tun, err := h.getUDPTun(ctx)
		if err != nil {
			h.logger.Error(err)
			return
		}
		defer tun.Close()

		go h.tunnelClientUDP(
			socks.NewUDPConn(relay, h.md.udpBufferSize),
			socks.UDPTunClientConn(tun, nil),
		)
	}

	t := time.Now()
	h.logger.Infof("%s <-> %s", conn.RemoteAddr(), &saddr)
	io.Copy(ioutil.Discard, conn)
	h.logger.
		WithFields(map[string]interface{}{"duration": time.Since(t)}).
		Infof("%s >-< %s", conn.RemoteAddr(), &saddr)
}

func (h *socks5Handler) getUDPTun(ctx context.Context) (conn net.Conn, err error) {
	r := (&handler.Router{}).
		WithChain(h.chain).
		WithRetry(h.md.retryCount).
		WithLogger(h.logger)
	conn, err = r.Connect(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			conn.Close()
			conn = nil
		}
	}()

	if h.md.timeout > 0 {
		conn.SetDeadline(time.Now().Add(h.md.timeout))
		defer conn.SetDeadline(time.Time{})
	}

	req := gosocks5.NewRequest(socks.CmdUDPTun, nil)
	if err = req.Write(conn); err != nil {
		return
	}
	h.logger.Debug(req)

	reply, err := gosocks5.ReadReply(conn)
	if err != nil {
		return
	}
	h.logger.Debug(reply)

	if reply.Rep != gosocks5.Succeeded {
		err = errors.New("UDP associate failed")
		return
	}

	return
}

func (h *socks5Handler) tunnelClientUDP(c, tun net.PacketConn) (err error) {
	bufSize := h.md.udpBufferSize
	errc := make(chan error, 2)

	go func() {
		for {
			err := func() error {
				b := bufpool.Get(bufSize)
				defer bufpool.Put(b)

				n, raddr, err := c.ReadFrom(b)
				if err != nil {
					return err
				}

				if h.bypass != nil && h.bypass.Contains(raddr.String()) {
					h.logger.Warn("bypass: ", raddr)
					return nil
				}

				if _, err := tun.WriteTo(b[:n], raddr); err != nil {
					return err
				}

				h.logger.Debugf("%s >>> %s data: %d",
					tun.LocalAddr(), raddr, n)

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

				n, raddr, err := tun.ReadFrom(b)
				if err != nil {
					return err
				}
				if h.bypass != nil && h.bypass.Contains(raddr.String()) {
					h.logger.Warn("bypass: ", raddr)
					return nil
				}

				if _, err := c.WriteTo(b[:n], raddr); err != nil {
					return err
				}

				h.logger.Debugf("%s <<< %s data: %d",
					tun.LocalAddr(), raddr, n)

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

func (h *socks5Handler) relayUDP(c, peer net.PacketConn) (err error) {
	bufSize := h.md.udpBufferSize
	errc := make(chan error, 2)

	go func() {
		for {
			err := func() error {
				b := bufpool.Get(bufSize)
				defer bufpool.Put(b)

				n, raddr, err := c.ReadFrom(b)
				if err != nil {
					return err
				}

				if h.bypass != nil && h.bypass.Contains(raddr.String()) {
					h.logger.Warn("bypass: ", raddr)
					return nil
				}

				if _, err := peer.WriteTo(b[:n], raddr); err != nil {
					return err
				}

				h.logger.Debugf("%s >>> %s data: %d",
					peer.LocalAddr(), raddr, n)

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

				n, raddr, err := peer.ReadFrom(b)
				if err != nil {
					return err
				}

				if h.bypass != nil && h.bypass.Contains(raddr.String()) {
					h.logger.Warn("bypass: ", raddr)
					return nil
				}

				if _, err := c.WriteTo(b[:n], raddr); err != nil {
					return err
				}

				h.logger.Debugf("%s <<< %s data: %d",
					peer.LocalAddr(), raddr, n)

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
