package http3

import (
	"strings"

	mdata "github.com/go-gost/gost/pkg/metadata"
)

const (
	defaultAuthorizePath = "/authorize"
	defaultPushPath      = "/push"
	defaultPullPath      = "/pull"
	defaultBacklog       = 128
)

type metadata struct {
	authorizePath string
	pushPath      string
	pullPath      string
	backlog       int
}

func (l *http3Listener) parseMetadata(md mdata.Metadata) (err error) {
	const (
		authorizePath = "authorizePath"
		pushPath      = "pushPath"
		pullPath      = "pullPath"

		backlog = "backlog"
	)

	l.md.authorizePath = mdata.GetString(md, authorizePath)
	if !strings.HasPrefix(l.md.authorizePath, "/") {
		l.md.authorizePath = defaultAuthorizePath
	}
	l.md.pushPath = mdata.GetString(md, pushPath)
	if !strings.HasPrefix(l.md.pushPath, "/") {
		l.md.pushPath = defaultPushPath
	}
	l.md.pullPath = mdata.GetString(md, pullPath)
	if !strings.HasPrefix(l.md.pullPath, "/") {
		l.md.pullPath = defaultPullPath
	}

	l.md.backlog = mdata.GetInt(md, backlog)
	if l.md.backlog <= 0 {
		l.md.backlog = defaultBacklog
	}

	return
}
