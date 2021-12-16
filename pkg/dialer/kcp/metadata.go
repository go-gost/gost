package kcp

import (
	"encoding/json"
	"time"

	kcp_util "github.com/go-gost/gost/pkg/common/util/kcp"
	md "github.com/go-gost/gost/pkg/metadata"
)

type metadata struct {
	handshakeTimeout time.Duration
	config           *kcp_util.Config
}

func (d *kcpDialer) parseMetadata(md md.Metadata) (err error) {
	const (
		config           = "config"
		handshakeTimeout = "handshakeTimeout"
	)

	if mm, _ := md.Get(config).(map[interface{}]interface{}); len(mm) > 0 {
		m := make(map[string]interface{})
		for k, v := range mm {
			if sk, ok := k.(string); ok {
				m[sk] = v
			}
		}
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

	d.md.handshakeTimeout = md.GetDuration(handshakeTimeout)
	return
}
