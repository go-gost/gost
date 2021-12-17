package mux

import (
	"crypto/tls"
	"time"

	tls_util "github.com/go-gost/gost/pkg/common/util/tls"
	md "github.com/go-gost/gost/pkg/metadata"
)

const (
	defaultBacklog = 128
)

type metadata struct {
	tlsConfig *tls.Config

	muxKeepAliveDisabled bool
	muxKeepAliveInterval time.Duration
	muxKeepAliveTimeout  time.Duration
	muxMaxFrameSize      int
	muxMaxReceiveBuffer  int
	muxMaxStreamBuffer   int

	backlog int
}

func (l *mtlsListener) parseMetadata(md md.Metadata) (err error) {
	const (
		certFile = "certFile"
		keyFile  = "keyFile"
		caFile   = "caFile"

		backlog = "backlog"

		muxKeepAliveDisabled = "muxKeepAliveDisabled"
		muxKeepAliveInterval = "muxKeepAliveInterval"
		muxKeepAliveTimeout  = "muxKeepAliveTimeout"
		muxMaxFrameSize      = "muxMaxFrameSize"
		muxMaxReceiveBuffer  = "muxMaxReceiveBuffer"
		muxMaxStreamBuffer   = "muxMaxStreamBuffer"
	)

	l.md.tlsConfig, err = tls_util.LoadServerConfig(
		md.GetString(certFile),
		md.GetString(keyFile),
		md.GetString(caFile),
	)
	if err != nil {
		return
	}

	l.md.backlog = md.GetInt(backlog)
	if l.md.backlog <= 0 {
		l.md.backlog = defaultBacklog
	}

	l.md.muxKeepAliveDisabled = md.GetBool(muxKeepAliveDisabled)
	l.md.muxKeepAliveInterval = md.GetDuration(muxKeepAliveInterval)
	l.md.muxKeepAliveTimeout = md.GetDuration(muxKeepAliveTimeout)
	l.md.muxMaxFrameSize = md.GetInt(muxMaxFrameSize)
	l.md.muxMaxReceiveBuffer = md.GetInt(muxMaxReceiveBuffer)
	l.md.muxMaxStreamBuffer = md.GetInt(muxMaxStreamBuffer)

	return
}
