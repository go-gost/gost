package main

import (
	// Register connectors
	_ "github.com/go-gost/gost/pkg/components/connector/http"
	_ "github.com/go-gost/gost/pkg/components/connector/ss"

	// Register dialers
	_ "github.com/go-gost/gost/pkg/components/dialer/tcp"

	// Register handlers
	_ "github.com/go-gost/gost/pkg/components/handler/http"
	_ "github.com/go-gost/gost/pkg/components/handler/ss"
	_ "github.com/go-gost/gost/pkg/components/handler/ssu"

	// Register listeners
	_ "github.com/go-gost/gost/pkg/components/listener/ftcp"
	_ "github.com/go-gost/gost/pkg/components/listener/http2"
	_ "github.com/go-gost/gost/pkg/components/listener/http2/h2"
	_ "github.com/go-gost/gost/pkg/components/listener/kcp"
	_ "github.com/go-gost/gost/pkg/components/listener/obfs/http"
	_ "github.com/go-gost/gost/pkg/components/listener/obfs/tls"
	_ "github.com/go-gost/gost/pkg/components/listener/quic"
	_ "github.com/go-gost/gost/pkg/components/listener/tcp"
	_ "github.com/go-gost/gost/pkg/components/listener/tls"
	_ "github.com/go-gost/gost/pkg/components/listener/tls/mux"
	_ "github.com/go-gost/gost/pkg/components/listener/udp"
	_ "github.com/go-gost/gost/pkg/components/listener/ws"
	_ "github.com/go-gost/gost/pkg/components/listener/ws/mux"
)
