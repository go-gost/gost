package v5

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"time"

	"github.com/go-gost/gosocks5"
	"github.com/go-gost/gost/pkg/common/util/socks"
	"github.com/go-gost/gost/pkg/handler"
	"github.com/go-gost/gost/pkg/logger"
)

func (h *socks5Handler) handleUDP(ctx context.Context, conn net.Conn, log logger.Logger) {
	log = log.WithFields(map[string]interface{}{
		"cmd": "udp",
	})

	if !h.md.enableUDP {
		reply := gosocks5.NewReply(gosocks5.NotAllowed, nil)
		reply.Write(conn)
		log.Debug(reply)
		log.Error("UDP relay is diabled")
		return
	}

	cc, err := net.ListenUDP("udp", nil)
	if err != nil {
		log.Error(err)
		reply := gosocks5.NewReply(gosocks5.Failure, nil)
		reply.Write(conn)
		log.Debug(reply)
		return
	}
	defer cc.Close()

	saddr := gosocks5.Addr{}
	saddr.ParseFrom(cc.LocalAddr().String())
	saddr.Type = 0
	saddr.Host, _, _ = net.SplitHostPort(conn.LocalAddr().String()) // replace the IP to the out-going interface's
	reply := gosocks5.NewReply(gosocks5.Succeeded, &saddr)
	if err := reply.Write(conn); err != nil {
		log.Error(err)
		return
	}
	log.Debug(reply)

	log = log.WithFields(map[string]interface{}{
		"bind": fmt.Sprintf("%s/%s", cc.LocalAddr(), cc.LocalAddr().Network()),
	})
	log.Debugf("bind on %s OK", cc.LocalAddr())

	// obtain a udp connection
	c, err := h.router.Dial(ctx, "udp", "") // UDP association
	if err != nil {
		log.Error(err)
		return
	}
	defer c.Close()

	pc, ok := c.(net.PacketConn)
	if !ok {
		log.Errorf("wrong connection type")
		return
	}

	relay := handler.NewUDPRelay(socks.UDPConn(cc, h.md.udpBufferSize), pc).
		WithBypass(h.options.Bypass).
		WithLogger(log)
	relay.SetBufferSize(h.md.udpBufferSize)

	go relay.Run()

	t := time.Now()
	log.Infof("%s <-> %s", conn.RemoteAddr(), cc.LocalAddr())
	io.Copy(ioutil.Discard, conn)
	log.WithFields(map[string]interface{}{"duration": time.Since(t)}).
		Infof("%s >-< %s", conn.RemoteAddr(), cc.LocalAddr())
}
