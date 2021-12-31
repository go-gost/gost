package impl

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/go-gost/gost/pkg/chain"
	resolver_util "github.com/go-gost/gost/pkg/internal/util/resolver"
	"github.com/go-gost/gost/pkg/logger"
	resolverpkg "github.com/go-gost/gost/pkg/resolver"
	"github.com/go-gost/gost/pkg/resolver/exchanger"
	"github.com/miekg/dns"
)

type NameServer struct {
	Addr      string
	Chain     *chain.Chain
	TTL       time.Duration
	Timeout   time.Duration
	ClientIP  net.IP
	Prefer    string
	Hostname  string // for TLS handshake verification
	exchanger exchanger.Exchanger
}

type resolverOptions struct {
	domain string
	logger logger.Logger
}

type ResolverOption func(opts *resolverOptions)

func DomainResolverOption(domain string) ResolverOption {
	return func(opts *resolverOptions) {
		opts.domain = domain
	}
}

func LoggerResolverOption(logger logger.Logger) ResolverOption {
	return func(opts *resolverOptions) {
		opts.logger = logger
	}
}

type resolver struct {
	servers []NameServer
	cache   *resolver_util.Cache
	options resolverOptions
	logger  logger.Logger
}

func NewResolver(nameservers []NameServer, opts ...ResolverOption) (resolverpkg.Resolver, error) {
	options := resolverOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	var servers []NameServer
	for _, server := range nameservers {
		addr := strings.TrimSpace(server.Addr)
		if addr == "" {
			continue
		}
		ex, err := exchanger.NewExchanger(
			addr,
			exchanger.RouterOption(&chain.Router{
				Chain:  server.Chain,
				Logger: options.logger,
			}),
			exchanger.TimeoutOption(server.Timeout),
			exchanger.LoggerOption(options.logger),
		)
		if err != nil {
			options.logger.Warnf("parse %s: %v", server, err)
			continue
		}

		server.exchanger = ex
		servers = append(servers, server)
	}
	cache := resolver_util.NewCache().
		WithLogger(options.logger)

	return &resolver{
		servers: servers,
		cache:   cache,
		options: options,
		logger:  options.logger,
	}, nil
}

func (r *resolver) Resolve(ctx context.Context, host string) (ips []net.IP, err error) {
	if ip := net.ParseIP(host); ip != nil {
		return []net.IP{ip}, nil
	}

	if r.options.domain != "" &&
		!strings.Contains(host, ".") {
		host = host + "." + r.options.domain
	}

	for _, server := range r.servers {
		ips, err = r.resolve(ctx, &server, host)
		if err != nil {
			r.logger.Error(err)
			continue
		}

		r.logger.Debugf("resolve %s via %s: %v", host, server.exchanger.String(), ips)

		if len(ips) > 0 {
			break
		}
	}

	return
}

func (r *resolver) resolve(ctx context.Context, server *NameServer, host string) (ips []net.IP, err error) {
	if server == nil {
		return
	}

	if server.Prefer == "ipv6" { // prefer ipv6
		mq := dns.Msg{}
		mq.SetQuestion(dns.Fqdn(host), dns.TypeAAAA)
		ips, err = r.resolveIPs(ctx, server, &mq)
		if err != nil || len(ips) > 0 {
			return
		}
	}

	// fallback to ipv4
	mq := dns.Msg{}
	mq.SetQuestion(dns.Fqdn(host), dns.TypeA)
	return r.resolveIPs(ctx, server, &mq)
}

func (r *resolver) resolveIPs(ctx context.Context, server *NameServer, mq *dns.Msg) (ips []net.IP, err error) {
	key := resolver_util.NewCacheKey(&mq.Question[0])
	mr := r.cache.Load(key)
	if mr == nil {
		resolver_util.AddSubnetOpt(mq, server.ClientIP)
		mr, err = r.exchange(ctx, server.exchanger, mq)
		if err != nil {
			return
		}
		r.cache.Store(key, mr, server.TTL)
	}

	for _, ans := range mr.Answer {
		if ar, _ := ans.(*dns.AAAA); ar != nil {
			ips = append(ips, ar.AAAA)
		}
		if ar, _ := ans.(*dns.A); ar != nil {
			ips = append(ips, ar.A)
		}
	}

	return
}

func (r *resolver) exchange(ctx context.Context, ex exchanger.Exchanger, mq *dns.Msg) (mr *dns.Msg, err error) {
	query, err := mq.Pack()
	if err != nil {
		return
	}
	reply, err := ex.Exchange(ctx, query)
	if err != nil {
		return
	}

	mr = &dns.Msg{}
	err = mr.Unpack(reply)

	return
}
