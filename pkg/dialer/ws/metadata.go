package ws

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	tls_util "github.com/go-gost/gost/pkg/common/util/tls"
	mdata "github.com/go-gost/gost/pkg/metadata"
)

const (
	defaultPath = "/ws"
)

type metadata struct {
	path      string
	host      string
	tlsConfig *tls.Config

	handshakeTimeout  time.Duration
	readHeaderTimeout time.Duration
	readBufferSize    int
	writeBufferSize   int
	enableCompression bool

	header http.Header
}

func (d *wsDialer) parseMetadata(md mdata.Metadata) (err error) {
	const (
		path = "path"
		host = "host"

		certFile   = "certFile"
		keyFile    = "keyFile"
		caFile     = "caFile"
		secure     = "secure"
		serverName = "serverName"

		handshakeTimeout  = "handshakeTimeout"
		readHeaderTimeout = "readHeaderTimeout"
		readBufferSize    = "readBufferSize"
		writeBufferSize   = "writeBufferSize"
		enableCompression = "enableCompression"

		header = "header"
	)

	d.md.path = mdata.GetString(md, path)
	if d.md.path == "" {
		d.md.path = defaultPath
	}

	d.md.host = mdata.GetString(md, host)

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
	d.md.readHeaderTimeout = mdata.GetDuration(md, readHeaderTimeout)
	d.md.readBufferSize = mdata.GetInt(md, readBufferSize)
	d.md.writeBufferSize = mdata.GetInt(md, writeBufferSize)
	d.md.enableCompression = mdata.GetBool(md, enableCompression)

	if m := mdata.GetStringMapString(md, header); len(m) > 0 {
		h := http.Header{}
		for k, v := range m {
			h.Add(k, v)
		}
		d.md.header = h
	}

	return
}
