package exchanger

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/logger"
	"github.com/miekg/dns"
)

type Options struct {
	chain     *chain.Chain
	tlsConfig *tls.Config
	timeout   time.Duration
	logger    logger.Logger
}

// Option allows a common way to set Exchanger options.
type Option func(opts *Options)

// ChainOption sets the chain for Exchanger.
func ChainOption(chain *chain.Chain) Option {
	return func(opts *Options) {
		opts.chain = chain
	}
}

// TLSConfigOption sets the TLS config for Exchanger.
func TLSConfigOption(cfg *tls.Config) Option {
	return func(opts *Options) {
		opts.tlsConfig = cfg
	}
}

// LoggerOption sets the logger for Exchanger.
func LoggerOption(logger logger.Logger) Option {
	return func(opts *Options) {
		opts.logger = logger
	}
}

// TimeoutOption sets the timeout for Exchanger.
func TimeoutOption(timeout time.Duration) Option {
	return func(opts *Options) {
		opts.timeout = timeout
	}
}

// Exchanger is an interface for DNS synchronous query.
type Exchanger interface {
	Exchange(ctx context.Context, msg []byte) ([]byte, error)
	String() string
}

type exchanger struct {
	network string
	addr    string
	rawAddr string
	router  *chain.Router
	client  *http.Client
	options Options
}

// NewExchanger create an Exchanger.
func NewExchanger(addr string, opts ...Option) (Exchanger, error) {
	var options Options
	for _, opt := range opts {
		opt(&options)
	}

	if !strings.Contains(addr, "://") {
		addr = "udp://" + addr
	}
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}

	ex := &exchanger{
		network: u.Scheme,
		addr:    u.Host,
		rawAddr: addr,
		options: options,
	}
	ex.router = (&chain.Router{}).
		WithChain(options.chain).
		WithLogger(options.logger)
	if _, port, _ := net.SplitHostPort(ex.addr); port == "" {
		ex.addr = net.JoinHostPort(ex.addr, "53")
	}

	switch ex.network {
	case "tcp":
	case "dot", "tls":
		if ex.options.tlsConfig == nil {
			ex.options.tlsConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
		}
		ex.network = "tcp"
	case "doh":
		ex.addr = addr
		if ex.options.tlsConfig == nil {
			ex.options.tlsConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
		}
		ex.client = &http.Client{
			Timeout: options.timeout,
			Transport: &http.Transport{
				TLSClientConfig:       options.tlsConfig,
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   options.timeout,
				ExpectContinueTimeout: 1 * time.Second,
				DialContext:           ex.dial,
			},
		}
	default:
		ex.network = "udp"
	}

	return ex, nil
}

func (ex *exchanger) Exchange(ctx context.Context, msg []byte) ([]byte, error) {
	if ex.network == "doh" {
		return ex.dohExchange(ctx, msg)
	}
	return ex.exchange(ctx, msg)
}

func (ex *exchanger) dohExchange(ctx context.Context, msg []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", ex.addr, bytes.NewBuffer(msg))
	if err != nil {
		return nil, fmt.Errorf("failed to create an HTTPS request: %w", err)
	}

	// req.Header.Add("Content-Type", "application/dns-udpwireformat")
	req.Header.Add("Content-Type", "application/dns-message")

	client := ex.client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform an HTTPS request: %w", err)
	}

	// Check response status code
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("returned status code %d", resp.StatusCode)
	}

	// Read wireformat response from the body
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read the response body: %w", err)
	}

	return buf, nil
}

func (ex *exchanger) exchange(ctx context.Context, msg []byte) ([]byte, error) {
	if ex.options.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, ex.options.timeout)
		defer cancel()
	}

	c, err := ex.dial(ctx, ex.network, ex.addr)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	if ex.options.tlsConfig != nil {
		c = tls.Client(c, ex.options.tlsConfig)
	}

	conn := &dns.Conn{Conn: c}

	if _, err = conn.Write(msg); err != nil {
		return nil, err
	}

	mr, err := conn.ReadMsg()
	if err != nil {
		return nil, err
	}

	return mr.Pack()
}

func (ex *exchanger) dial(ctx context.Context, network, address string) (net.Conn, error) {
	return ex.router.Dial(ctx, network, address)
}

func (ex *exchanger) String() string {
	return ex.rawAddr
}
