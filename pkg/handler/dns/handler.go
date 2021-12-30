package dns

import (
	"bytes"
	"context"
	"errors"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/go-gost/gost/pkg/bypass"
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
	bypass     bypass.Bypass
	router     *chain.Router
	exchangers []exchanger.Exchanger
	cache      *resolver_util.Cache
	logger     logger.Logger
	md         metadata
}

func NewHandler(opts ...handler.Option) handler.Handler {
	options := &handler.Options{}
	for _, opt := range opts {
		opt(options)
	}

	cache := resolver_util.NewCache().WithLogger(options.Logger)

	return &dnsHandler{
		bypass: options.Bypass,
		router: options.Router,
		cache:  cache,
		logger: options.Logger,
	}
}

func (h *dnsHandler) Init(md md.Metadata) (err error) {
	if err = h.parseMetadata(md); err != nil {
		return
	}

	for _, server := range h.md.servers {
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

	n, err := conn.Read(b)
	if err != nil {
		h.logger.Error(err)
		return
	}
	h.logger.Info("read data: ", n)

	reply, err := h.exchange(ctx, b[:n])
	if err != nil {
		return
	}

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
	// cache only for single question message.
	if len(mq.Question) == 1 {
		key := resolver_util.NewCacheKey(&mq.Question[0])
		mr = h.cache.Load(key)
		if mr != nil {
			h.logger.Debugf("exchange message %d (cached): %s", mq.Id, mq.Question[0].String())
			mr.Id = mq.Id
			return mr.Pack()
		}

		defer func() {
			if mr != nil {
				h.cache.Store(key, mr, h.md.ttl)
			}
		}()
	}

	query, err := mq.Pack()
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

func (h *dnsHandler) dumpMsgHeader(m *dns.Msg) string {
	buf := new(bytes.Buffer)
	buf.WriteString(m.MsgHdr.String() + " ")
	buf.WriteString("QUERY: " + strconv.Itoa(len(m.Question)) + ", ")
	buf.WriteString("ANSWER: " + strconv.Itoa(len(m.Answer)) + ", ")
	buf.WriteString("AUTHORITY: " + strconv.Itoa(len(m.Ns)) + ", ")
	buf.WriteString("ADDITIONAL: " + strconv.Itoa(len(m.Extra)))
	return buf.String()
}
