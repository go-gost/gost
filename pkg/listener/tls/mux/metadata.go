package mux

import (
	"crypto/tls"
	"time"

	tls_util "github.com/go-gost/gost/pkg/common/util/tls"
	mdata "github.com/go-gost/gost/pkg/metadata"
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

func (l *mtlsListener) parseMetadata(md mdata.Metadata) (err error) {
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
		mdata.GetString(md, certFile),
		mdata.GetString(md, keyFile),
		mdata.GetString(md, caFile),
	)
	if err != nil {
		return
	}

	l.md.backlog = mdata.GetInt(md, backlog)
	if l.md.backlog <= 0 {
		l.md.backlog = defaultBacklog
	}

	l.md.muxKeepAliveDisabled = mdata.GetBool(md, muxKeepAliveDisabled)
	l.md.muxKeepAliveInterval = mdata.GetDuration(md, muxKeepAliveInterval)
	l.md.muxKeepAliveTimeout = mdata.GetDuration(md, muxKeepAliveTimeout)
	l.md.muxMaxFrameSize = mdata.GetInt(md, muxMaxFrameSize)
	l.md.muxMaxReceiveBuffer = mdata.GetInt(md, muxMaxReceiveBuffer)
	l.md.muxMaxStreamBuffer = mdata.GetInt(md, muxMaxStreamBuffer)

	return
}
