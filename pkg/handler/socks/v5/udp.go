package v5

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"time"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/v3/pkg/common/net/relay"
	"github.com/go-gost/gost/v3/pkg/internal/util/socks"
	"github.com/go-gost/gost/v3/pkg/logger"
)

func (h *socks5Handler) handleUDP(ctx context.Context, conn net.Conn, log logger.Logger) error {
	log = log.WithFields(map[string]any{
		"cmd": "udp",
	})

	if !h.md.enableUDP {
		reply := gosocks5.NewReply(gosocks5.NotAllowed, nil)
		log.Debug(reply)
		log.Error("socks5: UDP relay is disabled")
		return reply.Write(conn)
	}

	cc, err := net.ListenUDP("udp", nil)
	if err != nil {
		log.Error(err)
		reply := gosocks5.NewReply(gosocks5.Failure, nil)
		reply.Write(conn)
		log.Debug(reply)
		return err
	}
	defer cc.Close()

	saddr := gosocks5.Addr{}
	saddr.ParseFrom(cc.LocalAddr().String())
	saddr.Type = 0
	saddr.Host, _, _ = net.SplitHostPort(conn.LocalAddr().String()) // replace the IP to the out-going interface's
	reply := gosocks5.NewReply(gosocks5.Succeeded, &saddr)
	if err := reply.Write(conn); err != nil {
		log.Error(err)
		return err
	}
	log.Debug(reply)

	log = log.WithFields(map[string]any{
		"bind": fmt.Sprintf("%s/%s", cc.LocalAddr(), cc.LocalAddr().Network()),
	})
	log.Debugf("bind on %s OK", cc.LocalAddr())

	// obtain a udp connection
	c, err := h.router.Dial(ctx, "udp", "") // UDP association
	if err != nil {
		log.Error(err)
		return err
	}
	defer c.Close()

	pc, ok := c.(net.PacketConn)
	if !ok {
		err := errors.New("socks5: wrong connection type")
		log.Error(err)
		return err
	}

	r := relay.NewUDPRelay(socks.UDPConn(cc, h.md.udpBufferSize), pc).
		WithBypass(h.options.Bypass).
		WithLogger(log)
	r.SetBufferSize(h.md.udpBufferSize)

	go r.Run()

	t := time.Now()
	log.Infof("%s <-> %s", conn.RemoteAddr(), cc.LocalAddr())
	io.Copy(ioutil.Discard, conn)
	log.WithFields(map[string]any{"duration": time.Since(t)}).
		Infof("%s >-< %s", conn.RemoteAddr(), cc.LocalAddr())

	return nil
}
