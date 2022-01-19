package dns

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/common/bufpool"
	"github.com/go-gost/gost/pkg/handler"
	resolver_util "github.com/go-gost/gost/pkg/internal/util/resolver"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/go-gost/gost/pkg/resolver/exchanger"
	"github.com/miekg/dns"
)

const (
	defaultNameserver = "udp://127.0.0.1:53"
)

func init() {
	registry.RegisterHandler("dns", NewHandler)
}

type dnsHandler struct {
	exchangers []exchanger.Exchanger
	cache      *resolver_util.Cache
	router     *chain.Router
	logger     logger.Logger
	md         metadata
	options    handler.Options
}

func NewHandler(opts ...handler.Option) handler.Handler {
	options := handler.Options{}
	for _, opt := range opts {
		opt(&options)
	}

	return &dnsHandler{
		options: options,
	}
}

func (h *dnsHandler) Init(md md.Metadata) (err error) {
	h.logger = h.options.Logger

	if err = h.parseMetadata(md); err != nil {
		return
	}

	h.cache = resolver_util.NewCache().WithLogger(h.options.Logger)
	h.router = &chain.Router{
		Retries:  h.options.Retries,
		Chain:    h.options.Chain,
		Resolver: h.options.Resolver,
		// Hosts:    h.options.Hosts,
		Logger: h.options.Logger,
	}

	for _, server := range h.md.dns {
		server = strings.TrimSpace(server)
		if server == "" {
			continue
		}
		ex, err := exchanger.NewExchanger(
			server,
			exchanger.RouterOption(h.router),
			exchanger.TimeoutOption(h.md.timeout),
			exchanger.LoggerOption(h.logger),
		)
		if err != nil {
			h.logger.Warnf("parse %s: %v", server, err)
			continue
		}
		h.exchangers = append(h.exchangers, ex)
	}
	if len(h.exchangers) == 0 {
		ex, err := exchanger.NewExchanger(
			defaultNameserver,
			exchanger.RouterOption(h.router),
			exchanger.TimeoutOption(h.md.timeout),
			exchanger.LoggerOption(h.logger),
		)
		h.logger.Warnf("resolver not found, default to %s", defaultNameserver)
		if err != nil {
			return err
		}
		h.exchangers = append(h.exchangers, ex)
	}

	return
}

func (h *dnsHandler) Handle(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	start := time.Now()
	h.logger = h.logger.WithFields(map[string]interface{}{
		"remote": conn.RemoteAddr().String(),
		"local":  conn.LocalAddr().String(),
	})

	h.logger.Infof("%s <> %s", conn.RemoteAddr(), conn.LocalAddr())
	defer func() {
		h.logger.WithFields(map[string]interface{}{
			"duration": time.Since(start),
		}).Infof("%s >< %s", conn.RemoteAddr(), conn.LocalAddr())
	}()

	b := bufpool.Get(4096)
	defer bufpool.Put(b)

	n, err := conn.Read(*b)
	if err != nil {
		h.logger.Error(err)
		return
	}
	h.logger.Info("read data: ", n)

	reply, err := h.exchange(ctx, (*b)[:n])
	if err != nil {
		return
	}
	defer bufpool.Put(&reply)

	if _, err = conn.Write(reply); err != nil {
		h.logger.Error(err)
	}
}

func (h *dnsHandler) exchange(ctx context.Context, msg []byte) ([]byte, error) {
	mq := dns.Msg{}
	if err := mq.Unpack(msg); err != nil {
		h.logger.Error(err)
		return nil, err
	}

	if len(mq.Question) == 0 {
		return nil, errors.New("msg: empty question")
	}

	resolver_util.AddSubnetOpt(&mq, h.md.clientIP)

	if h.logger.IsLevelEnabled(logger.DebugLevel) {
		h.logger.Debug(mq.String())
	} else {
		h.logger.Info(h.dumpMsgHeader(&mq))
	}

	var mr *dns.Msg

	if h.logger.IsLevelEnabled(logger.DebugLevel) {
		defer func() {
			if mr != nil {
				h.logger.Debug(mr.String())
			}
		}()
	}

	mr = h.lookupHosts(&mq)
	if mr != nil {
		b := bufpool.Get(4096)
		return mr.PackBuffer(*b)
	}

	// only cache for single question message.
	if len(mq.Question) == 1 {
		key := resolver_util.NewCacheKey(&mq.Question[0])
		mr = h.cache.Load(key)
		if mr != nil {
			h.logger.Debugf("exchange message %d (cached): %s", mq.Id, mq.Question[0].String())
			mr.Id = mq.Id

			b := bufpool.Get(4096)
			return mr.PackBuffer(*b)
		}

		defer func() {
			if mr != nil {
				h.cache.Store(key, mr, h.md.ttl)
			}
		}()
	}

	b := bufpool.Get(4096)
	defer bufpool.Put(b)

	query, err := mq.PackBuffer(*b)
	if err != nil {
		h.logger.Error(err)
		return nil, err
	}

	var reply []byte
	for _, ex := range h.exchangers {
		h.logger.Infof("exchange message %d via %s: %s", mq.Id, ex.String(), mq.Question[0].String())
		reply, err = ex.Exchange(ctx, query)
		if err == nil {
			break
		}
		h.logger.Error(err)
	}
	if err != nil {
		return nil, err
	}

	mr = &dns.Msg{}
	if err = mr.Unpack(reply); err != nil {
		h.logger.Error(err)
		return nil, err
	}

	if h.logger.IsLevelEnabled(logger.DebugLevel) {
		h.logger.Debug(mr.String())
	} else {
		h.logger.Info(h.dumpMsgHeader(mr))
	}

	return reply, nil
}

// lookup host mapper
func (h *dnsHandler) lookupHosts(r *dns.Msg) (m *dns.Msg) {
	if h.options.Hosts == nil ||
		r.Question[0].Qclass != dns.ClassINET ||
		(r.Question[0].Qtype != dns.TypeA && r.Question[0].Qtype != dns.TypeAAAA) {
		return nil
	}

	m = &dns.Msg{}
	m.SetReply(r)

	host := strings.TrimSuffix(r.Question[0].Name, ".")

	switch r.Question[0].Qtype {
	case dns.TypeA:
		ips, _ := h.options.Hosts.Lookup("ip4", host)
		if len(ips) == 0 {
			return nil
		}
		h.logger.Debugf("hit host mapper: %s -> %s", host, ips)

		for _, ip := range ips {
			rr, err := dns.NewRR(fmt.Sprintf("%s IN A %s\n", r.Question[0].Name, ip.String()))
			if err != nil {
				h.logger.Error(err)
				return nil
			}
			m.Answer = append(m.Answer, rr)
		}

	case dns.TypeAAAA:
		ips, _ := h.options.Hosts.Lookup("ip6", host)
		if len(ips) == 0 {
			return nil
		}
		h.logger.Debugf("hit host mapper: %s -> %s", host, ips)

		for _, ip := range ips {
			rr, err := dns.NewRR(fmt.Sprintf("%s IN AAAA %s\n", r.Question[0].Name, ip.String()))
			if err != nil {
				h.logger.Error(err)
				return nil
			}
			m.Answer = append(m.Answer, rr)
		}
	}

	return
}

func (h *dnsHandler) dumpMsgHeader(m *dns.Msg) string {
	buf := new(bytes.Buffer)
	buf.WriteString(m.MsgHdr.String() + " ")
	buf.WriteString("QUERY: " + strconv.Itoa(len(m.Question)) + ", ")
	buf.WriteString("ANSWER: " + strconv.Itoa(len(m.Answer)) + ", ")
	buf.WriteString("AUTHORITY: " + strconv.Itoa(len(m.Ns)) + ", ")
	buf.WriteString("ADDITIONAL: " + strconv.Itoa(len(m.Extra)))
	return buf.String()
}
