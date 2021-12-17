package mux

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	tls_util "github.com/go-gost/gost/pkg/common/util/tls"
	md "github.com/go-gost/gost/pkg/metadata"
)

const (
	defaultPath = "/ws"
)

type metadata struct {
	path      string
	host      string
	tlsConfig *tls.Config

	handshakeTimeout  time.Duration
	readHeaderTimeout time.Duration
	readBufferSize    int
	writeBufferSize   int
	enableCompression bool

	muxKeepAliveDisabled bool
	muxKeepAliveInterval time.Duration
	muxKeepAliveTimeout  time.Duration
	muxMaxFrameSize      int
	muxMaxReceiveBuffer  int
	muxMaxStreamBuffer   int

	header http.Header
}

func (d *mwsDialer) parseMetadata(md md.Metadata) (err error) {
	const (
		path = "path"
		host = "host"

		certFile   = "certFile"
		keyFile    = "keyFile"
		caFile     = "caFile"
		secure     = "secure"
		serverName = "serverName"

		handshakeTimeout  = "handshakeTimeout"
		readHeaderTimeout = "readHeaderTimeout"
		readBufferSize    = "readBufferSize"
		writeBufferSize   = "writeBufferSize"
		enableCompression = "enableCompression"

		header = "header"

		muxKeepAliveDisabled = "muxKeepAliveDisabled"
		muxKeepAliveInterval = "muxKeepAliveInterval"
		muxKeepAliveTimeout  = "muxKeepAliveTimeout"
		muxMaxFrameSize      = "muxMaxFrameSize"
		muxMaxReceiveBuffer  = "muxMaxReceiveBuffer"
		muxMaxStreamBuffer   = "muxMaxStreamBuffer"
	)

	d.md.path = md.GetString(path)
	if d.md.path == "" {
		d.md.path = defaultPath
	}

	d.md.host = md.GetString(host)

	sn, _, _ := net.SplitHostPort(md.GetString(serverName))
	if sn == "" {
		sn = "localhost"
	}
	d.md.tlsConfig, err = tls_util.LoadClientConfig(
		md.GetString(certFile),
		md.GetString(keyFile),
		md.GetString(caFile),
		md.GetBool(secure),
		sn,
	)

	d.md.muxKeepAliveDisabled = md.GetBool(muxKeepAliveDisabled)
	d.md.muxKeepAliveInterval = md.GetDuration(muxKeepAliveInterval)
	d.md.muxKeepAliveTimeout = md.GetDuration(muxKeepAliveTimeout)
	d.md.muxMaxFrameSize = md.GetInt(muxMaxFrameSize)
	d.md.muxMaxReceiveBuffer = md.GetInt(muxMaxReceiveBuffer)
	d.md.muxMaxStreamBuffer = md.GetInt(muxMaxStreamBuffer)

	d.md.handshakeTimeout = md.GetDuration(handshakeTimeout)
	d.md.readHeaderTimeout = md.GetDuration(readHeaderTimeout)
	d.md.readBufferSize = md.GetInt(readBufferSize)
	d.md.writeBufferSize = md.GetInt(writeBufferSize)
	d.md.enableCompression = md.GetBool(enableCompression)

	if mm, _ := md.Get(header).(map[interface{}]interface{}); len(mm) > 0 {
		h := http.Header{}
		for k, v := range mm {
			h.Add(fmt.Sprintf("%v", k), fmt.Sprintf("%v", v))
		}
		d.md.header = h
	}
	return
}
