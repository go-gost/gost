package v5

import (
	"bytes"
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
	"github.com/go-gost/gost/pkg/logger"
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
		if h.logger.IsLevelEnabled(logger.DebugLevel) {
			h.logger.Debug(reply)
		}
		return
	}
	defer relay.Close()

	saddr, _ := gosocks5.NewAddr(relay.LocalAddr().String())
	if saddr == nil {
		saddr = &gosocks5.Addr{}
	}
	saddr.Type = 0
	saddr.Host, _, _ = net.SplitHostPort(conn.LocalAddr().String()) // replace the IP to the out-going interface's
	reply := gosocks5.NewReply(gosocks5.Succeeded, saddr)
	if err := reply.Write(conn); err != nil {
		h.logger.Error(err)
		return
	}
	if h.logger.IsLevelEnabled(logger.DebugLevel) {
		h.logger.Debug(reply)
	}

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

		go h.relayUDP(relay, peer)
	} else {
		tun, err := h.getUDPTun(ctx)
		if err != nil {
			h.logger.Error(err)
			return
		}
		defer tun.Close()

		go h.tunnelClientUDP(relay, tun)
	}

	t := time.Now()
	h.logger.Infof("%s <-> %s", conn.RemoteAddr(), saddr)
	io.Copy(ioutil.Discard, conn)
	h.logger.
		WithFields(map[string]interface{}{"duration": time.Since(t)}).
		Infof("%s >-< %s", conn.RemoteAddr(), saddr)
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
	if h.logger.IsLevelEnabled(logger.DebugLevel) {
		h.logger.Debug(req)
	}

	reply, err := gosocks5.ReadReply(conn)
	if err != nil {
		return
	}
	if h.logger.IsLevelEnabled(logger.DebugLevel) {
		h.logger.Debug(reply)
	}

	if reply.Rep != gosocks5.Succeeded {
		err = errors.New("UDP associate failed")
		return
	}

	return
}

func (h *socks5Handler) tunnelClientUDP(c net.PacketConn, tunnel net.Conn) (err error) {
	bufSize := h.md.udpBufferSize
	errc := make(chan error, 2)

	var clientAddr net.Addr

	go func() {
		b := bufpool.Get(bufSize)
		defer bufpool.Put(b)

		for {
			n, laddr, err := c.ReadFrom(b)
			if err != nil {
				errc <- err
				return
			}

			if clientAddr == nil {
				clientAddr = laddr
			}

			var addr gosocks5.Addr
			header := gosocks5.UDPHeader{
				Addr: &addr,
			}
			hlen, err := header.ReadFrom(bytes.NewReader(b[:n]))
			if err != nil {
				errc <- err
				return
			}

			raddr, err := net.ResolveUDPAddr("udp", addr.String())
			if err != nil {
				continue // drop silently
			}

			if h.bypass != nil && h.bypass.Contains(raddr.String()) {
				h.logger.Warn("bypass: ", raddr)
				continue // bypass
			}

			dgram := gosocks5.UDPDatagram{
				Header: &header,
				Data:   b[hlen:n],
			}
			dgram.Header.Rsv = uint16(len(dgram.Data))

			if _, err := dgram.WriteTo(tunnel); err != nil {
				errc <- err
				return
			}

			if h.logger.IsLevelEnabled(logger.DebugLevel) {
				h.logger.Debugf("%s >>> %s: %v data: %d",
					clientAddr, raddr, b[:hlen], len(dgram.Data))
			}
		}
	}()

	go func() {
		b := bufpool.Get(bufSize)
		defer bufpool.Put(b)

		const dataPos = 262

		for {
			addr := gosocks5.Addr{}
			header := gosocks5.UDPHeader{
				Addr: &addr,
			}

			data := b[dataPos:]
			dgram := gosocks5.UDPDatagram{
				Header: &header,
				Data:   data,
			}
			_, err := dgram.ReadFrom(tunnel)
			if err != nil {
				errc <- err
				return
			}
			// NOTE: the dgram.Data may be reallocated if the provided buffer is too short,
			// we drop it for simplicity. As this occurs, you should enlarge the buffer size.
			if len(dgram.Data) > len(data) {
				h.logger.Warnf("buffer too short, dropped")
				continue
			}

			// pipe from tunnel to relay
			if clientAddr == nil {
				h.logger.Warnf("ignore unexpected peer from %s", addr)
				continue
			}

			raddr := addr.String()
			if h.bypass != nil && h.bypass.Contains(raddr) {
				h.logger.Warn("bypass: ", raddr)
				continue // bypass
			}

			addrLen := addr.Length()
			addr.Encode(b[dataPos-addrLen : dataPos])

			hlen := addrLen + 3
			if _, err := c.WriteTo(b[dataPos-hlen:dataPos+len(dgram.Data)], clientAddr); err != nil {
				errc <- err
				return
			}

			if h.logger.IsLevelEnabled(logger.DebugLevel) {
				h.logger.Debugf("%s <<< %s: %v data: %d",
					clientAddr, addr.String(), b[dataPos-hlen:dataPos], len(dgram.Data))
			}
		}
	}()

	return <-errc
}

func (h *socks5Handler) relayUDP(c, peer net.PacketConn) (err error) {
	bufSize := h.md.udpBufferSize
	errc := make(chan error, 2)

	var clientAddr net.Addr

	go func() {
		b := bufpool.Get(bufSize)
		defer bufpool.Put(b)

		for {
			n, laddr, err := c.ReadFrom(b)
			if err != nil {
				errc <- err
				return
			}
			if clientAddr == nil {
				clientAddr = laddr
			}

			var addr gosocks5.Addr
			header := gosocks5.UDPHeader{
				Addr: &addr,
			}
			hlen, err := header.ReadFrom(bytes.NewReader(b[:n]))
			if err != nil {
				errc <- err
				return
			}

			raddr, err := net.ResolveUDPAddr("udp", addr.String())
			if err != nil {
				continue // drop silently
			}

			if h.bypass != nil && h.bypass.Contains(raddr.String()) {
				h.logger.Warn("bypass: ", raddr)
				continue // bypass
			}

			data := b[hlen:n]
			if _, err := peer.WriteTo(data, raddr); err != nil {
				errc <- err
				return
			}
			if h.logger.IsLevelEnabled(logger.DebugLevel) {
				h.logger.Debugf("%s >>> %s: %v data: %d",
					clientAddr, raddr, b[:hlen], len(data))
			}
		}
	}()

	go func() {
		b := bufpool.Get(bufSize)
		defer bufpool.Put(b)

		const dataPos = 262

		for {
			n, raddr, err := peer.ReadFrom(b[dataPos:])
			if err != nil {
				errc <- err
				return
			}
			if clientAddr == nil {
				continue
			}
			if h.bypass != nil && h.bypass.Contains(raddr.String()) {
				h.logger.Warn("bypass: ", raddr)
				continue // bypass
			}

			socksAddr, _ := gosocks5.NewAddr(raddr.String())
			if socksAddr == nil {
				socksAddr = &gosocks5.Addr{}
			}
			addrLen := socksAddr.Length()
			socksAddr.Encode(b[dataPos-addrLen : dataPos])

			hlen := addrLen + 3
			if _, err := c.WriteTo(b[dataPos-hlen:dataPos+n], clientAddr); err != nil {
				errc <- err
				return
			}

			if h.logger.IsLevelEnabled(logger.DebugLevel) {
				h.logger.Debugf("%s <<< %s: %v data: %d",
					clientAddr, raddr, b[dataPos-hlen:dataPos], n)
			}
		}
	}()

	return <-errc
}
