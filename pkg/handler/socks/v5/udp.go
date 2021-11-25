package v5

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"time"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/pkg/common/bufpool"
	"github.com/go-gost/gost/pkg/common/util/socks"
)

func (h *socks5Handler) handleUDP(ctx context.Context, conn net.Conn) {
	h.logger = h.logger.WithFields(map[string]interface{}{
		"cmd": "udp",
	})

	if !h.md.enableUDP {
		reply := gosocks5.NewReply(gosocks5.NotAllowed, nil)
		reply.Write(conn)
		h.logger.Debug(reply)
		h.logger.Error("UDP relay is diabled")
		return
	}

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
		"bind": fmt.Sprintf("%s/%s", relay.LocalAddr(), relay.LocalAddr().Network()),
	})
	h.logger.Debugf("bind on %s OK", relay.LocalAddr())

	peer, err := net.ListenUDP("udp", nil)
	if err != nil {
		h.logger.Error(err)
		return
	}
	defer peer.Close()

	go h.relayUDP(
		socks.UDPConn(relay, h.md.udpBufferSize),
		peer,
	)

	t := time.Now()
	h.logger.Infof("%s <-> %s", conn.RemoteAddr(), relay.LocalAddr())
	io.Copy(ioutil.Discard, conn)
	h.logger.
		WithFields(map[string]interface{}{"duration": time.Since(t)}).
		Infof("%s >-< %s", conn.RemoteAddr(), relay.LocalAddr())
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
