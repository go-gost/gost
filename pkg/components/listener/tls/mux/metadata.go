package mux

import (
	"crypto/tls"
	"time"
)

const (
	addr     = "addr"
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

const (
	defaultQueueSize = 128
)

type metadata struct {
	addr      string
	tlsConfig *tls.Config

	muxKeepAliveDisabled bool
	muxKeepAlivePeriod   time.Duration
	muxKeepAliveTimeout  time.Duration
	muxMaxFrameSize      int
	muxMaxReceiveBuffer  int
	muxMaxStreamBuffer   int

	connQueueSize int
}
