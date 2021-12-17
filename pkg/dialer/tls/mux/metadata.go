package mux

import (
	"crypto/tls"
	"net"
	"time"

	tls_util "github.com/go-gost/gost/pkg/common/util/tls"
	md "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	tlsConfig        *tls.Config
	handshakeTimeout time.Duration

	muxKeepAliveDisabled bool
	muxKeepAliveInterval time.Duration
	muxKeepAliveTimeout  time.Duration
	muxMaxFrameSize      int
	muxMaxReceiveBuffer  int
	muxMaxStreamBuffer   int
}

func (d *mtlsDialer) parseMetadata(md md.Metadata) (err error) {
	const (
		certFile   = "certFile"
		keyFile    = "keyFile"
		caFile     = "caFile"
		secure     = "secure"
		serverName = "serverName"

		handshakeTimeout = "handshakeTimeout"

		muxKeepAliveDisabled = "muxKeepAliveDisabled"
		muxKeepAliveInterval = "muxKeepAliveInterval"
		muxKeepAliveTimeout  = "muxKeepAliveTimeout"
		muxMaxFrameSize      = "muxMaxFrameSize"
		muxMaxReceiveBuffer  = "muxMaxReceiveBuffer"
		muxMaxStreamBuffer   = "muxMaxStreamBuffer"
	)

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
	d.md.handshakeTimeout = md.GetDuration(handshakeTimeout)

	d.md.muxKeepAliveDisabled = md.GetBool(muxKeepAliveDisabled)
	d.md.muxKeepAliveInterval = md.GetDuration(muxKeepAliveInterval)
	d.md.muxKeepAliveTimeout = md.GetDuration(muxKeepAliveTimeout)
	d.md.muxMaxFrameSize = md.GetInt(muxMaxFrameSize)
	d.md.muxMaxReceiveBuffer = md.GetInt(muxMaxReceiveBuffer)
	d.md.muxMaxStreamBuffer = md.GetInt(muxMaxStreamBuffer)

	return
}
