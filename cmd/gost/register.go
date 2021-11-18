package main

import (
	// Register connectors
	_ "github.com/go-gost/gost/pkg/connector/http"
	_ "github.com/go-gost/gost/pkg/connector/socks/v4"
	_ "github.com/go-gost/gost/pkg/connector/socks/v5"
	_ "github.com/go-gost/gost/pkg/connector/ss"

	// Register dialers
	_ "github.com/go-gost/gost/pkg/dialer/tcp"
	_ "github.com/go-gost/gost/pkg/dialer/udp"

	// Register handlers
	_ "github.com/go-gost/gost/pkg/handler/forward"
	_ "github.com/go-gost/gost/pkg/handler/http"
	_ "github.com/go-gost/gost/pkg/handler/relay"
	_ "github.com/go-gost/gost/pkg/handler/socks/v4"
	_ "github.com/go-gost/gost/pkg/handler/socks/v5"
	_ "github.com/go-gost/gost/pkg/handler/ss"

	// Register listeners
	_ "github.com/go-gost/gost/pkg/listener/ftcp"
	_ "github.com/go-gost/gost/pkg/listener/http2"
	_ "github.com/go-gost/gost/pkg/listener/http2/h2"
	_ "github.com/go-gost/gost/pkg/listener/kcp"
	_ "github.com/go-gost/gost/pkg/listener/obfs/http"
	_ "github.com/go-gost/gost/pkg/listener/obfs/tls"
	_ "github.com/go-gost/gost/pkg/listener/quic"
	_ "github.com/go-gost/gost/pkg/listener/rtcp"
	_ "github.com/go-gost/gost/pkg/listener/rudp"
	_ "github.com/go-gost/gost/pkg/listener/tcp"
	_ "github.com/go-gost/gost/pkg/listener/tls"
	_ "github.com/go-gost/gost/pkg/listener/tls/mux"
	_ "github.com/go-gost/gost/pkg/listener/udp"
	_ "github.com/go-gost/gost/pkg/listener/ws"
	_ "github.com/go-gost/gost/pkg/listener/ws/mux"
)
