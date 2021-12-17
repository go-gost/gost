package ws

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	tls_util "github.com/go-gost/gost/pkg/common/util/tls"
	md "github.com/go-gost/gost/pkg/metadata"
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

func (l *wsListener) parseMetadata(md md.Metadata) (err error) {
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
		md.GetString(certFile),
		md.GetString(keyFile),
		md.GetString(caFile),
	)
	if err != nil {
		return
	}

	l.md.path = md.GetString(path)
	if l.md.path == "" {
		l.md.path = defaultPath
	}

	l.md.backlog = md.GetInt(backlog)
	if l.md.backlog <= 0 {
		l.md.backlog = defaultBacklog
	}

	l.md.handshakeTimeout = md.GetDuration(handshakeTimeout)
	l.md.readHeaderTimeout = md.GetDuration(readHeaderTimeout)
	l.md.readBufferSize = md.GetInt(readBufferSize)
	l.md.writeBufferSize = md.GetInt(writeBufferSize)
	l.md.enableCompression = md.GetBool(enableCompression)

	if mm, _ := md.Get(header).(map[interface{}]interface{}); len(mm) > 0 {
		h := http.Header{}
		for k, v := range mm {
			h.Add(fmt.Sprintf("%v", k), fmt.Sprintf("%v", v))
		}
		l.md.header = h
	}

	return
}
