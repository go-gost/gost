package mux

import (
	"crypto/tls"
	"net"
	"time"

	tls_util "github.com/go-gost/gost/pkg/common/util/tls"
	mdata "github.com/go-gost/gost/pkg/metadata"
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

func (d *mtlsDialer) parseMetadata(md mdata.Metadata) (err error) {
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

	sn, _, _ := net.SplitHostPort(mdata.GetString(md, serverName))
	if sn == "" {
		sn = "localhost"
	}
	d.md.tlsConfig, err = tls_util.LoadClientConfig(
		mdata.GetString(md, certFile),
		mdata.GetString(md, keyFile),
		mdata.GetString(md, caFile),
		mdata.GetBool(md, secure),
		sn,
	)
	d.md.handshakeTimeout = mdata.GetDuration(md, handshakeTimeout)

	d.md.muxKeepAliveDisabled = mdata.GetBool(md, muxKeepAliveDisabled)
	d.md.muxKeepAliveInterval = mdata.GetDuration(md, muxKeepAliveInterval)
	d.md.muxKeepAliveTimeout = mdata.GetDuration(md, muxKeepAliveTimeout)
	d.md.muxMaxFrameSize = mdata.GetInt(md, muxMaxFrameSize)
	d.md.muxMaxReceiveBuffer = mdata.GetInt(md, muxMaxReceiveBuffer)
	d.md.muxMaxStreamBuffer = mdata.GetInt(md, muxMaxStreamBuffer)

	return
}
