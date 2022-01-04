package mux

import (
	"time"

	mdata "github.com/go-gost/gost/pkg/metadata"
)

const (
	defaultBacklog = 128
)

type metadata struct {
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
		backlog = "backlog"

		muxKeepAliveDisabled = "muxKeepAliveDisabled"
		muxKeepAliveInterval = "muxKeepAliveInterval"
		muxKeepAliveTimeout  = "muxKeepAliveTimeout"
		muxMaxFrameSize      = "muxMaxFrameSize"
		muxMaxReceiveBuffer  = "muxMaxReceiveBuffer"
		muxMaxStreamBuffer   = "muxMaxStreamBuffer"
	)

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
