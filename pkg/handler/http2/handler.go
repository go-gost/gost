package http2

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-gost/gost/pkg/auth"
	"github.com/go-gost/gost/pkg/chain"
	auth_util "github.com/go-gost/gost/pkg/common/util/auth"
	"github.com/go-gost/gost/pkg/handler"
	http2_util "github.com/go-gost/gost/pkg/internal/util/http2"
	"github.com/go-gost/gost/pkg/logger"
	md "github.com/go-gost/gost/pkg/metadata"
	"github.com/go-gost/gost/pkg/registry"
)

func init() {
	registry.RegisterHandler("http2", NewHandler)
}

type http2Handler struct {
	router        *chain.Router
	authenticator auth.Authenticator
	logger        logger.Logger
	md            metadata
	options       handler.Options
}

func NewHandler(opts ...handler.Option) handler.Handler {
	options := handler.Options{}
	for _, opt := range opts {
		opt(&options)
	}

	return &http2Handler{
		options: options,
	}
}

func (h *http2Handler) Init(md md.Metadata) error {
	if err := h.parseMetadata(md); err != nil {
		return err
	}

	h.authenticator = auth_util.AuthFromUsers(h.options.Auths...)
	h.router = &chain.Router{
		Retries:  h.options.Retries,
		Chain:    h.options.Chain,
		Resolver: h.options.Resolver,
		Hosts:    h.options.Hosts,
		Logger:   h.options.Logger,
	}
	h.logger = h.options.Logger
	return nil
}

func (h *http2Handler) Handle(ctx context.Context, conn net.Conn) {
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

	cc, ok := conn.(*http2_util.ServerConn)
	if !ok {
		h.logger.Error("wrong connection type")
		return
	}
	h.roundTrip(ctx, cc.Writer(), cc.Request())
}

// NOTE: there is an issue (golang/go#43989) will cause the client hangs
// when server returns an non-200 status code,
// May be fixed in go1.18.
func (h *http2Handler) roundTrip(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	// Try to get the actual host.
	// Compatible with GOST 2.x.
	if v := req.Header.Get("Gost-Target"); v != "" {
		if h, err := h.decodeServerName(v); err == nil {
			req.Host = h
		}
	}
	req.Header.Del("Gost-Target")

	if v := req.Header.Get("X-Gost-Target"); v != "" {
		if h, err := h.decodeServerName(v); err == nil {
			req.Host = h
		}
	}
	req.Header.Del("X-Gost-Target")

	addr := req.Host
	if _, port, _ := net.SplitHostPort(addr); port == "" {
		addr = net.JoinHostPort(addr, "80")
	}

	fields := map[string]interface{}{
		"dst": addr,
	}
	if u, _, _ := h.basicProxyAuth(req.Header.Get("Proxy-Authorization")); u != "" {
		fields["user"] = u
	}
	h.logger = h.logger.WithFields(fields)

	if h.logger.IsLevelEnabled(logger.DebugLevel) {
		dump, _ := httputil.DumpRequest(req, false)
		h.logger.Debug(string(dump))
	}
	h.logger.Infof("%s >> %s", req.RemoteAddr, addr)

	if h.md.proxyAgent != "" {
		w.Header().Set("Proxy-Agent", h.md.proxyAgent)
	}

	if h.options.Bypass != nil && h.options.Bypass.Contains(addr) {
		w.WriteHeader(http.StatusForbidden)
		h.logger.Info("bypass: ", addr)
		return
	}

	resp := &http.Response{
		ProtoMajor: 2,
		ProtoMinor: 0,
		Header:     http.Header{},
		Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
	}

	if !h.authenticate(w, req, resp) {
		return
	}

	// delete the proxy related headers.
	req.Header.Del("Proxy-Authorization")
	req.Header.Del("Proxy-Connection")

	cc, err := h.router.Dial(ctx, "tcp", addr)
	if err != nil {
		h.logger.Error(err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	defer cc.Close()

	if req.Method == http.MethodConnect {
		w.WriteHeader(http.StatusOK)
		if fw, ok := w.(http.Flusher); ok {
			fw.Flush()
		}

		// compatible with HTTP1.x
		if hj, ok := w.(http.Hijacker); ok && req.ProtoMajor == 1 {
			// we take over the underly connection
			conn, _, err := hj.Hijack()
			if err != nil {
				h.logger.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			defer conn.Close()

			start := time.Now()
			h.logger.Infof("%s <-> %s", conn.RemoteAddr(), addr)
			handler.Transport(conn, cc)
			h.logger.
				WithFields(map[string]interface{}{
					"duration": time.Since(start),
				}).
				Infof("%s >-< %s", conn.RemoteAddr(), addr)
		}

		start := time.Now()
		h.logger.Infof("%s <-> %s", req.RemoteAddr, addr)
		handler.Transport(&readWriter{r: req.Body, w: flushWriter{w}}, cc)
		h.logger.
			WithFields(map[string]interface{}{
				"duration": time.Since(start),
			}).
			Infof("%s >-< %s", req.RemoteAddr, addr)
		return
	}
}

func (h *http2Handler) decodeServerName(s string) (string, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	if len(b) < 4 {
		return "", errors.New("invalid name")
	}
	v, err := base64.RawURLEncoding.DecodeString(string(b[4:]))
	if err != nil {
		return "", err
	}
	if crc32.ChecksumIEEE(v) != binary.BigEndian.Uint32(b[:4]) {
		return "", errors.New("invalid name")
	}
	return string(v), nil
}

func (h *http2Handler) basicProxyAuth(proxyAuth string) (username, password string, ok bool) {
	if proxyAuth == "" {
		return
	}

	if !strings.HasPrefix(proxyAuth, "Basic ") {
		return
	}
	c, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(proxyAuth, "Basic "))
	if err != nil {
		return
	}
	cs := string(c)
	s := strings.IndexByte(cs, ':')
	if s < 0 {
		return
	}

	return cs[:s], cs[s+1:], true
}

func (h *http2Handler) authenticate(w http.ResponseWriter, r *http.Request, resp *http.Response) (ok bool) {
	u, p, _ := h.basicProxyAuth(r.Header.Get("Proxy-Authorization"))
	if h.authenticator == nil || h.authenticator.Authenticate(u, p) {
		return true
	}

	pr := h.md.probeResistance
	// probing resistance is enabled, and knocking host is mismatch.
	if pr != nil && (pr.Knock == "" || !strings.EqualFold(r.URL.Hostname(), pr.Knock)) {
		resp.StatusCode = http.StatusServiceUnavailable // default status code
		switch pr.Type {
		case "code":
			resp.StatusCode, _ = strconv.Atoi(pr.Value)
		case "web":
			url := pr.Value
			if !strings.HasPrefix(url, "http") {
				url = "http://" + url
			}
			r, err := http.Get(url)
			if err != nil {
				h.logger.Error(err)
				break
			}
			resp = r
			defer resp.Body.Close()
		case "host":
			cc, err := net.Dial("tcp", pr.Value)
			if err != nil {
				h.logger.Error(err)
				break
			}
			defer cc.Close()

			if err := h.forwardRequest(w, r, cc); err != nil {
				h.logger.Error(err)
			}
			return
		case "file":
			f, _ := os.Open(pr.Value)
			if f != nil {
				defer f.Close()

				resp.StatusCode = http.StatusOK
				if finfo, _ := f.Stat(); finfo != nil {
					resp.ContentLength = finfo.Size()
				}
				resp.Header.Set("Content-Type", "text/html")
				resp.Body = f
			}
		}
	}

	if resp.StatusCode == 0 {
		resp.StatusCode = http.StatusProxyAuthRequired
		resp.Header.Add("Proxy-Authenticate", "Basic realm=\"gost\"")
		if strings.ToLower(r.Header.Get("Proxy-Connection")) == "keep-alive" {
			// XXX libcurl will keep sending auth request in same conn
			// which we don't supported yet.
			resp.Header.Add("Connection", "close")
			resp.Header.Add("Proxy-Connection", "close")
		}

		h.logger.Info("proxy authentication required")
	} else {
		resp.Header = http.Header{}
		resp.Header.Set("Server", "nginx/1.20.1")
		resp.Header.Set("Date", time.Now().Format(http.TimeFormat))
		if resp.StatusCode == http.StatusOK {
			resp.Header.Set("Connection", "keep-alive")
		}
	}

	if h.logger.IsLevelEnabled(logger.DebugLevel) {
		dump, _ := httputil.DumpResponse(resp, false)
		h.logger.Debug(string(dump))
	}

	h.writeResponse(w, resp)

	return
}
func (h *http2Handler) forwardRequest(w http.ResponseWriter, r *http.Request, rw io.ReadWriter) (err error) {
	if err = r.Write(rw); err != nil {
		return
	}

	resp, err := http.ReadResponse(bufio.NewReader(rw), r)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	return h.writeResponse(w, resp)
}

func (h *http2Handler) writeResponse(w http.ResponseWriter, resp *http.Response) error {
	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, err := io.Copy(flushWriter{w}, resp.Body)
	return err
}
