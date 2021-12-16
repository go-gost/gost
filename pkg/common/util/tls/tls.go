package tls

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"net"
	"time"
)

var (
	// DefaultConfig is a default TLS config for global use.
	DefaultConfig *tls.Config
)

// LoadServerConfig loads the certificate from cert & key files and optional client CA file.
func LoadServerConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	if certFile == "" && keyFile == "" {
		return DefaultConfig.Clone(), nil
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	cfg := &tls.Config{Certificates: []tls.Certificate{cert}}

	pool, err := loadCA(caFile)
	if err != nil {
		return nil, err
	}
	if pool != nil {
		cfg.ClientCAs = pool
		cfg.ClientAuth = tls.RequireAndVerifyClientCert
	}

	return cfg, nil
}

// LoadClientConfig loads the certificate from cert & key files and optional CA file.
func LoadClientConfig(certFile, keyFile, caFile string, verify bool, serverName string) (*tls.Config, error) {
	var cfg *tls.Config

	if certFile == "" && keyFile == "" {
		cfg = &tls.Config{}
	} else {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, err
		}

		cfg = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	}

	rootCAs, err := loadCA(caFile)
	if err != nil {
		return nil, err
	}

	cfg.RootCAs = rootCAs
	cfg.ServerName = serverName
	cfg.InsecureSkipVerify = !verify

	// If the root ca is given, but skip verify, we verify the certificate manually.
	if cfg.RootCAs != nil && !verify {
		cfg.VerifyConnection = func(state tls.ConnectionState) error {
			opts := x509.VerifyOptions{
				Roots:         cfg.RootCAs,
				CurrentTime:   time.Now(),
				DNSName:       "",
				Intermediates: x509.NewCertPool(),
			}

			certs := state.PeerCertificates
			for i, cert := range certs {
				if i == 0 {
					continue
				}
				opts.Intermediates.AddCert(cert)
			}

			_, err := certs[0].Verify(opts)
			return err
		}
	}

	return cfg, nil
}

func loadCA(caFile string) (cp *x509.CertPool, err error) {
	if caFile == "" {
		return
	}
	cp = x509.NewCertPool()
	data, err := ioutil.ReadFile(caFile)
	if err != nil {
		return nil, err
	}
	if !cp.AppendCertsFromPEM(data) {
		return nil, errors.New("AppendCertsFromPEM failed")
	}
	return
}

// Wrap a net.Conn into a client tls connection, performing any
// additional verification as needed.
//
// As of go 1.3, crypto/tls only supports either doing no certificate
// verification, or doing full verification including of the peer's
// DNS name. For consul, we want to validate that the certificate is
// signed by a known CA, but because consul doesn't use DNS names for
// node names, we don't verify the certificate DNS names. Since go 1.3
// no longer supports this mode of operation, we have to do it
// manually.
//
// This code is taken from consul:
// https://github.com/hashicorp/consul/blob/master/tlsutil/config.go
func WrapTLSClient(conn net.Conn, tlsConfig *tls.Config, timeout time.Duration) (net.Conn, error) {
	var err error
	var tlsConn *tls.Conn

	if timeout > 0 {
		conn.SetDeadline(time.Now().Add(timeout))
		defer conn.SetDeadline(time.Time{})
	}

	tlsConn = tls.Client(conn, tlsConfig)

	// Otherwise perform handshake, but don't verify the domain
	//
	// The following is lightly-modified from the doFullHandshake
	// method in https://golang.org/src/crypto/tls/handshake_client.go
	if err = tlsConn.Handshake(); err != nil {
		tlsConn.Close()
		return nil, err
	}

	// We can do this in `tls.Config.VerifyConnection`, which effective for
	// other TLS protocols such as WebSocket. See `route.go:parseChainNode`
	/*
		// If crypto/tls is doing verification, there's no need to do our own.
		if tlsConfig.InsecureSkipVerify == false {
			return tlsConn, nil
		}

		// Similarly if we use host's CA, we can do full handshake
		if tlsConfig.RootCAs == nil {
			return tlsConn, nil
		}

		opts := x509.VerifyOptions{
			Roots:         tlsConfig.RootCAs,
			CurrentTime:   time.Now(),
			DNSName:       "",
			Intermediates: x509.NewCertPool(),
		}

		certs := tlsConn.ConnectionState().PeerCertificates
		for i, cert := range certs {
			if i == 0 {
				continue
			}
			opts.Intermediates.AddCert(cert)
		}

		_, err = certs[0].Verify(opts)
		if err != nil {
			tlsConn.Close()
			return nil, err
		}
	*/

	return tlsConn, err
}
