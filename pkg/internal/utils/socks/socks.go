package socks

const (
	// MethodTLS is an extended SOCKS5 method with tls encryption support.
	MethodTLS uint8 = 0x80
	// MethodTLSAuth is an extended SOCKS5 method with tls encryption and authentication support.
	MethodTLSAuth uint8 = 0x82
	// MethodMux is an extended SOCKS5 method for stream multiplexing.
	MethodMux = 0x88
)

const (
	// CmdMuxBind is an extended SOCKS5 request CMD for
	// multiplexing transport with the binding server.
	CmdMuxBind uint8 = 0xF2
	// CmdUDPTun is an extended SOCKS5 request CMD for UDP over TCP.
	CmdUDPTun uint8 = 0xF3
)
