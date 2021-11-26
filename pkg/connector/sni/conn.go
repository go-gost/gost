package sni

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"hash/crc32"
	"io"
	"net"
	"strings"

	dissector "github.com/go-gost/tls-dissector"
)

type sniClientConn struct {
	host       string
	obfuscated bool
	net.Conn
}

func (c *sniClientConn) Write(p []byte) (int, error) {
	b, err := c.obfuscate(p)
	if err != nil {
		return 0, err
	}
	if _, err = c.Conn.Write(b); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (c *sniClientConn) obfuscate(p []byte) ([]byte, error) {
	if c.host == "" {
		return p, nil
	}

	if c.obfuscated {
		return p, nil
	}

	if p[0] == dissector.Handshake {
		b, err := readClientHelloRecord(bytes.NewReader(p), c.host)
		if err != nil {
			return nil, err
		}
		c.obfuscated = true
		return b, nil
	}

	buf := &bytes.Buffer{}
	br := bufio.NewReader(bytes.NewReader(p))
	for {
		s, err := br.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				return nil, err
			}
			if s != "" {
				buf.Write([]byte(s))
			}
			break
		}

		// end of HTTP header
		if s == "\r\n" {
			buf.Write([]byte(s))
			// drain the remain bytes.
			io.Copy(buf, br)
			break
		}

		if strings.HasPrefix(s, "Host") {
			s = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(s, "Host:"), "\r\n"))
			host := encodeServerName(s)
			buf.WriteString("Host: " + c.host + "\r\n")
			buf.WriteString("Gost-Target: " + host + "\r\n")
			// drain the remain bytes.
			io.Copy(buf, br)
			break
		}
		buf.Write([]byte(s))
	}
	c.obfuscated = true
	return buf.Bytes(), nil
}

func readClientHelloRecord(r io.Reader, host string) ([]byte, error) {
	record, err := dissector.ReadRecord(r)
	if err != nil {
		return nil, err
	}
	clientHello := dissector.ClientHelloMsg{}
	if err := clientHello.Decode(record.Opaque); err != nil {
		return nil, err
	}

	for _, ext := range clientHello.Extensions {
		if ext.Type() == dissector.ExtServerName {
			snExtension := ext.(*dissector.ServerNameExtension)
			if host != "" {
				e, _ := dissector.NewExtension(0xFFFE, []byte(encodeServerName(snExtension.Name)))
				clientHello.Extensions = append(clientHello.Extensions, e)
				snExtension.Name = host
			}

			break
		}
	}
	record.Opaque, err = clientHello.Encode()
	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	if _, err := record.WriteTo(buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func encodeServerName(name string) string {
	buf := &bytes.Buffer{}
	binary.Write(buf, binary.BigEndian, crc32.ChecksumIEEE([]byte(name)))
	buf.WriteString(base64.RawURLEncoding.EncodeToString([]byte(name)))
	return base64.RawURLEncoding.EncodeToString(buf.Bytes())
}
