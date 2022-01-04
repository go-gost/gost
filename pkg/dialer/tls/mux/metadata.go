package mux

import (
	"time"

	mdata "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
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
		handshakeTimeout = "handshakeTimeout"

		muxKeepAliveDisabled = "muxKeepAliveDisabled"
		muxKeepAliveInterval = "muxKeepAliveInterval"
		muxKeepAliveTimeout  = "muxKeepAliveTimeout"
		muxMaxFrameSize      = "muxMaxFrameSize"
		muxMaxReceiveBuffer  = "muxMaxReceiveBuffer"
		muxMaxStreamBuffer   = "muxMaxStreamBuffer"
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
