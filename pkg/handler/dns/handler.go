package dns

import (
	"bytes"
	"context"
	"errors"
	"net"
	"strconv"
	"time"

	"github.com/go-gost/gost/pkg/bypass"
	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/common/bufpool"
	"github.com/go-gost/gost/pkg/handler"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
	"github.com/go-gost/gost/pkg/resolver/exchanger"
	"github.com/miekg/dns"
)

func init() {
	registry.RegisterHandler("dns", NewHandler)
}

type dnsHandler struct {
	chain      *chain.Chain
	bypass     bypass.Bypass
	exchangers []exchanger.Exchanger
	logger     logger.Logger
	md         metadata
}

func NewHandler(opts ...handler.Option) handler.Handler {
	options := &handler.Options{}
	for _, opt := range opts {
		opt(options)
	}

	return &dnsHandler{
		bypass: options.Bypass,
		logger: options.Logger,
	}
}

func (h *dnsHandler) Init(md md.Metadata) (err error) {
	if err = h.parseMetadata(md); err != nil {
		return
	}

	for _, server := range h.md.servers {
		ex, err := exchanger.NewExchanger(
			server,
			exchanger.ChainOption(h.chain),
			exchanger.LoggerOption(h.logger),
		)
		if err != nil {
			h.logger.Warnf("parse %s: %v", server, err)
			continue
		}
		h.exchangers = append(h.exchangers, ex)
	}
	if len(h.exchangers) == 0 {
		ex, _ := exchanger.NewExchanger(
			"udp://127.0.0.53:53",
			exchanger.ChainOption(h.chain),
			exchanger.LoggerOption(h.logger),
		)
		if ex != nil {
			h.exchangers = append(h.exchangers, ex)
		}
	}
	return
}

// implements chain.Chainable interface
func (h *dnsHandler) WithChain(chain *chain.Chain) {
	h.chain = chain
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
		h.logger.Error(err)
		return
	}

	if _, err = conn.Write(reply); err != nil {
		h.logger.Error(err)
	}
}

func (h *dnsHandler) exchange(ctx context.Context, msg []byte) ([]byte, error) {
	mq := dns.Msg{}
	if err := mq.Unpack(msg); err != nil {
		return nil, err
	}

	if len(mq.Question) == 0 {
		return nil, errors.New("msg: empty question")
	}

	if h.logger.IsLevelEnabled(logger.DebugLevel) {
		h.logger.Debug(mq.String())
	} else {
		h.logger.Info(h.dumpMsgHeader(&mq))
	}

	var mr *dns.Msg
	// Only cache for single question.
	/*
		if len(mq.Question) == 1 {
			key := newResolverCacheKey(&mq.Question[0])
			mr = r.cache.loadCache(key)
			if mr != nil {
				log.Logf("[dns] exchange message %d (cached): %s", mq.Id, mq.Question[0].String())
				mr.Id = mq.Id
				return mr.Pack()
			}

			defer func() {
				if mr != nil {
					r.cache.storeCache(key, mr, r.TTL())
				}
			}()
		}
	*/

	// r.addSubnetOpt(mq)

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
		h.logger.Error(err)
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
