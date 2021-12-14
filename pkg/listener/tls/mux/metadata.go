package mux

import (
	"crypto/tls"
	"time"

	tls_util "github.com/go-gost/gost/pkg/common/util/tls"
	md "github.com/go-gost/gost/pkg/metadata"
)

const (
	defaultQueueSize = 128
)

type metadata struct {
	tlsConfig *tls.Config

	muxKeepAliveDisabled bool
	muxKeepAlivePeriod   time.Duration
	muxKeepAliveTimeout  time.Duration
	muxMaxFrameSize      int
	muxMaxReceiveBuffer  int
	muxMaxStreamBuffer   int

	connQueueSize int
}

func (l *mtlsListener) parseMetadata(md md.Metadata) (err error) {
	const (
		certFile = "certFile"
		keyFile  = "keyFile"
		caFile   = "caFile"

		muxKeepAliveDisabled = "muxKeepAliveDisabled"
		muxKeepAlivePeriod   = "muxKeepAlivePeriod"
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

	return
}
