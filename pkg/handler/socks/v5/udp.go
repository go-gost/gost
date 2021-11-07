package v5

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net"
	"time"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/pkg/internal/bufpool"
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

	if !h.chain.IsEmpty() {

	}

	peer, err := net.ListenUDP("udp", nil)
	if err != nil {
		h.logger.Error(err)
		return
	}
	defer peer.Close()

	go h.transportUDP(relay, peer)

	t := time.Now()
	h.logger.Infof("%s <-> %s", conn.RemoteAddr(), saddr)
	io.Copy(ioutil.Discard, conn)
	h.logger.
		WithFields(map[string]interface{}{"duration": time.Since(t)}).
		Infof("%s >-< %s", conn.RemoteAddr(), saddr)
}

func (h *socks5Handler) transportUDP(relay, peer net.PacketConn) (err error) {
	const bufSize = 65 * 1024
	errc := make(chan error, 2)

	var clientAddr net.Addr

	go func() {
		b := bufpool.Get(bufSize)
		defer bufpool.Put(b)

		for {
			n, laddr, err := relay.ReadFrom(b)
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

		const dataPos = 1024

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
			socksAddr.Type = 0
			addrLen := socksAddr.Length()
			socksAddr.Encode(b[dataPos-addrLen : dataPos])

			hlen := addrLen + 3
			if _, err := relay.WriteTo(b[dataPos-hlen:dataPos+n], clientAddr); err != nil {
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
