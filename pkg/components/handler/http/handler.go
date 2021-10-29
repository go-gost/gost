package http

import (
	"bufio"
	"context"
	"net"
	"net/http"

	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/components/handler"
	"github.com/go-gost/gost/pkg/logger"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.RegisterHandler("http", NewHandler)
}

type Handler struct {
	chain  *chain.Chain
	logger logger.Logger
	md     metadata
}

func NewHandler(opts ...handler.Option) handler.Handler {
	options := &handler.Options{}
	for _, opt := range opts {
		opt(options)
	}

	return &Handler{
		chain:  options.Chain,
		logger: options.Logger,
	}
}

func (h *Handler) Init(md handler.Metadata) error {
	return nil
}

func (h *Handler) Handle(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	req, err := http.ReadRequest(bufio.NewReader(conn))
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"src":   conn.RemoteAddr(),
			"local": conn.LocalAddr(),
		}).Error(err)
		return
	}
	defer req.Body.Close()

	h.handleRequest(ctx, conn, req)
}

func (h *Handler) handleRequest(ctx context.Context, conn net.Conn, req *http.Request) {
	if req == nil {
		return
	}

	/*
		// try to get the actual host.
		if v := req.Header.Get("Gost-Target"); v != "" {
			if h, err := decodeServerName(v); err == nil {
				req.Host = h
			}
		}
	*/

	host := req.Host
	if _, port, _ := net.SplitHostPort(host); port == "" {
		host = net.JoinHostPort(host, "80")
	}

	/*
		u, _, _ := basicProxyAuth(req.Header.Get("Proxy-Authorization"))
		if u != "" {
			u += "@"
		}
		log.Logf("[http] %s%s -> %s -> %s",
			u, conn.RemoteAddr(), h.options.Node.String(), host)

		if Debug {
			dump, _ := httputil.DumpRequest(req, false)
			log.Logf("[http] %s -> %s\n%s", conn.RemoteAddr(), conn.LocalAddr(), string(dump))
		}

		req.Header.Del("Gost-Target")
	*/
	resp := &http.Response{
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header{},
	}

	if h.md.proxyAgent != "" {
		resp.Header.Add("Proxy-Agent", h.md.proxyAgent)
	}

	/*
		if !Can("tcp", host, h.options.Whitelist, h.options.Blacklist) {
			log.Logf("[http] %s - %s : Unauthorized to tcp connect to %s",
				conn.RemoteAddr(), conn.LocalAddr(), host)
			resp.StatusCode = http.StatusForbidden

			if Debug {
				dump, _ := httputil.DumpResponse(resp, false)
				log.Logf("[http] %s <- %s\n%s", conn.RemoteAddr(), conn.LocalAddr(), string(dump))
			}

			resp.Write(conn)
			return
		}
	*/

	/*
		if h.options.Bypass.Contains(host) {
			resp.StatusCode = http.StatusForbidden

			log.Logf("[http] %s - %s bypass %s",
				conn.RemoteAddr(), conn.LocalAddr(), host)
			if Debug {
				dump, _ := httputil.DumpResponse(resp, false)
				log.Logf("[http] %s <- %s\n%s", conn.RemoteAddr(), conn.LocalAddr(), string(dump))
			}

			resp.Write(conn)
			return
		}
	*/

	/*
		if !h.authenticate(conn, req, resp) {
			return
		}
	*/

	if req.Method == "PRI" ||
		(req.Method != http.MethodConnect && req.URL.Scheme != "http") {
		resp.StatusCode = http.StatusBadRequest
		/*
			if Debug {
				dump, _ := httputil.DumpResponse(resp, false)
				log.Logf("[http] %s <- %s\n%s",
					conn.RemoteAddr(), conn.LocalAddr(), string(dump))
			}
		*/

		resp.Write(conn)
		return
	}

	req.Header.Del("Proxy-Authorization")

	cc, err := h.dial(ctx, host)
	if err != nil {
		resp.StatusCode = http.StatusServiceUnavailable

		/*
			if Debug {
				dump, _ := httputil.DumpResponse(resp, false)
				log.Logf("[http] %s <- %s\n%s", conn.RemoteAddr(), conn.LocalAddr(), string(dump))
			}
		*/
		resp.Write(conn)
		return
	}
	defer cc.Close()

	if req.Method == http.MethodConnect {
		resp.StatusCode = http.StatusOK
		resp.Status = "200 Connection established"
		resp.Write(conn)
	} else {
		req.Header.Del("Proxy-Connection")

		if err = req.Write(cc); err != nil {
			return
		}
	}

	handler.Transport(conn, cc)
}

func (h *Handler) dial(ctx context.Context, addr string) (conn net.Conn, err error) {
	count := h.md.retryCount + 1
	if count <= 0 {
		count = 1
	}

	for i := 0; i < count; i++ {
		route := h.chain.GetRoute()

		/*
			buf := bytes.Buffer{}
			fmt.Fprintf(&buf, "%s -> %s -> ",
				conn.RemoteAddr(), h.options.Node.String())
			for _, nd := range route.route {
				fmt.Fprintf(&buf, "%d@%s -> ", nd.ID, nd.String())
			}
			fmt.Fprintf(&buf, "%s", host)
			log.Log("[route]", buf.String())
		*/

		/*
			// forward http request
			lastNode := route.LastNode()
			if req.Method != http.MethodConnect && lastNode.Protocol == "http" {
				err = h.forwardRequest(conn, req, route)
				if err == nil {
					return
				}
				log.Logf("[http] %s -> %s : %s", conn.RemoteAddr(), conn.LocalAddr(), err)
				continue
			}
		*/

		conn, err = route.Dial(ctx, "tcp", addr)
		if err != nil {
			continue
		}
	}

	return
}
