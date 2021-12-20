package kcp

import (
	"encoding/json"
	"time"

	kcp_util "github.com/go-gost/gost/pkg/common/util/kcp"
	mdata "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	handshakeTimeout time.Duration
	config           *kcp_util.Config
}

func (d *kcpDialer) parseMetadata(md mdata.Metadata) (err error) {
	const (
		config           = "config"
		handshakeTimeout = "handshakeTimeout"
	)

	if m := mdata.GetStringMap(md, config); len(m) > 0 {
		b, err := json.Marshal(m)
		if err != nil {
			return err
		}
		cfg := &kcp_util.Config{}
		if err := json.Unmarshal(b, cfg); err != nil {
			return err
		}
		d.md.config = cfg
	}
	if d.md.config == nil {
		d.md.config = kcp_util.DefaultConfig
	}

	d.md.handshakeTimeout = mdata.GetDuration(md, handshakeTimeout)
	return
}
