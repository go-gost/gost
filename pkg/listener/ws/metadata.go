package ws

import (
	"crypto/tls"
	"net/http"
	"time"

	tls_util "github.com/go-gost/gost/pkg/common/util/tls"
	mdata "github.com/go-gost/gost/pkg/metadata"
)

const (
	defaultPath    = "/ws"
	defaultBacklog = 128
)

type metadata struct {
	path      string
	backlog   int
	tlsConfig *tls.Config

	handshakeTimeout  time.Duration
	readHeaderTimeout time.Duration
	readBufferSize    int
	writeBufferSize   int
	enableCompression bool

	header http.Header
}

func (l *wsListener) parseMetadata(md mdata.Metadata) (err error) {
	const (
		certFile = "certFile"
		keyFile  = "keyFile"
		caFile   = "caFile"

		path    = "path"
		backlog = "backlog"

		handshakeTimeout  = "handshakeTimeout"
		readHeaderTimeout = "readHeaderTimeout"
		readBufferSize    = "readBufferSize"
		writeBufferSize   = "writeBufferSize"
		enableCompression = "enableCompression"

		header = "header"
	)

	l.md.tlsConfig, err = tls_util.LoadServerConfig(
		mdata.GetString(md, certFile),
		mdata.GetString(md, keyFile),
		mdata.GetString(md, caFile),
	)
	if err != nil {
		return
	}

	l.md.path = mdata.GetString(md, path)
	if l.md.path == "" {
		l.md.path = defaultPath
	}

	l.md.backlog = mdata.GetInt(md, backlog)
	if l.md.backlog <= 0 {
		l.md.backlog = defaultBacklog
	}

	l.md.handshakeTimeout = mdata.GetDuration(md, handshakeTimeout)
	l.md.readHeaderTimeout = mdata.GetDuration(md, readHeaderTimeout)
	l.md.readBufferSize = mdata.GetInt(md, readBufferSize)
	l.md.writeBufferSize = mdata.GetInt(md, writeBufferSize)
	l.md.enableCompression = mdata.GetBool(md, enableCompression)

	if mm := mdata.GetStringMapString(md, header); len(mm) > 0 {
		hd := http.Header{}
		for k, v := range mm {
			hd.Add(k, v)
		}
		l.md.header = hd
	}
	return
}
