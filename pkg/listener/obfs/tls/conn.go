package tls

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"net"
	"sync"
	"time"

	dissector "github.com/go-gost/tls-dissector"
)

const (
	maxTLSDataLen = 16384
)

type obfsTLSConn struct {
	net.Conn
	rbuf           bytes.Buffer
	wbuf           bytes.Buffer
	handshaked     bool
	handshakeMutex sync.Mutex
}

func (c *obfsTLSConn) Handshake() (err error) {
	c.handshakeMutex.Lock()
	defer c.handshakeMutex.Unlock()

	if c.handshaked {
		return
	}

	if err = c.handshake(); err != nil {
		return
	}

	c.handshaked = true
	return nil
}

func (c *obfsTLSConn) handshake() error {
	record := &dissector.Record{}
	if _, err := record.ReadFrom(c.Conn); err != nil {
		// log.Log(err)
		return err
	}
	if record.Type != dissector.Handshake {
		return dissector.ErrBadType
	}

	clientMsg := &dissector.ClientHelloMsg{}
	if err := clientMsg.Decode(record.Opaque); err != nil {
		// log.Log(err)
		return err
	}

	for _, ext := range clientMsg.Extensions {
		if ext.Type() == dissector.ExtSessionTicket {
			b, err := ext.Encode()
			if err != nil {
				// log.Log(err)
				return err
			}
			c.rbuf.Write(b)
			break
		}
	}

	serverMsg := &dissector.ServerHelloMsg{
		Version:           tls.VersionTLS12,
		SessionID:         clientMsg.SessionID,
		CipherSuite:       0xcca8,
		CompressionMethod: 0x00,
		Extensions: []dissector.Extension{
			&dissector.RenegotiationInfoExtension{},
			&dissector.ExtendedMasterSecretExtension{},
			&dissector.ECPointFormatsExtension{
				Formats: []uint8{0x00},
			},
		},
	}

	serverMsg.Random.Time = uint32(time.Now().Unix())
	rand.Read(serverMsg.Random.Opaque[:])
	b, err := serverMsg.Encode()
	if err != nil {
		return err
	}

	record = &dissector.Record{
		Type:    dissector.Handshake,
		Version: tls.VersionTLS10,
		Opaque:  b,
	}

	if _, err := record.WriteTo(&c.wbuf); err != nil {
		return err
	}

	record = &dissector.Record{
		Type:    dissector.ChangeCipherSpec,
		Version: tls.VersionTLS12,
		Opaque:  []byte{0x01},
	}
	if _, err := record.WriteTo(&c.wbuf); err != nil {
		return err
	}
	return nil
}

func (c *obfsTLSConn) Read(b []byte) (n int, err error) {
	if err = c.Handshake(); err != nil {
		return
	}

	if c.rbuf.Len() > 0 {
		return c.rbuf.Read(b)
	}
	record := &dissector.Record{}
	if _, err = record.ReadFrom(c.Conn); err != nil {
		return
	}
	n = copy(b, record.Opaque)
	_, err = c.rbuf.Write(record.Opaque[n:])
	return
}

func (c *obfsTLSConn) Write(b []byte) (n int, err error) {
	if err = c.Handshake(); err != nil {
		return
	}
	n = len(b)

	for len(b) > 0 {
		data := b
		if len(b) > maxTLSDataLen {
			data = b[:maxTLSDataLen]
			b = b[maxTLSDataLen:]
		} else {
			b = b[:0]
		}
		record := &dissector.Record{
			Type:    dissector.AppData,
			Version: tls.VersionTLS12,
			Opaque:  data,
		}

		if c.wbuf.Len() > 0 {
			record.Type = dissector.Handshake
			record.WriteTo(&c.wbuf)
			_, err = c.wbuf.WriteTo(c.Conn)
			return
		}

		if _, err = record.WriteTo(c.Conn); err != nil {
			return
		}
	}
	return
}
