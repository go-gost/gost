package tls

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"errors"
	"net"
	"sync"
	"time"

	dissector "github.com/ginuerzh/tls-dissector"
)

const (
	maxTLSDataLen = 16384
)

var (
	cipherSuites = []uint16{
		0xc02c, 0xc030, 0x009f, 0xcca9, 0xcca8, 0xccaa, 0xc02b, 0xc02f,
		0x009e, 0xc024, 0xc028, 0x006b, 0xc023, 0xc027, 0x0067, 0xc00a,
		0xc014, 0x0039, 0xc009, 0xc013, 0x0033, 0x009d, 0x009c, 0x003d,
		0x003c, 0x0035, 0x002f, 0x00ff,
	}

	compressionMethods = []uint8{0x00}

	algorithms = []uint16{
		0x0601, 0x0602, 0x0603, 0x0501, 0x0502, 0x0503, 0x0401, 0x0402,
		0x0403, 0x0301, 0x0302, 0x0303, 0x0201, 0x0202, 0x0203,
	}

	tlsRecordTypes   = []uint8{0x16, 0x14, 0x16, 0x17}
	tlsVersionMinors = []uint8{0x01, 0x03, 0x03, 0x03}

	ErrBadType         = errors.New("bad type")
	ErrBadMajorVersion = errors.New("bad major version")
	ErrBadMinorVersion = errors.New("bad minor version")
	ErrMaxDataLen      = errors.New("bad tls data len")
)

const (
	tlsRecordStateType = iota
	tlsRecordStateVersion0
	tlsRecordStateVersion1
	tlsRecordStateLength0
	tlsRecordStateLength1
	tlsRecordStateData
)

type obfsTLSParser struct {
	step   uint8
	state  uint8
	length uint16
}

func (r *obfsTLSParser) Parse(b []byte) (int, error) {
	i := 0
	last := 0
	length := len(b)

	for i < length {
		ch := b[i]
		switch r.state {
		case tlsRecordStateType:
			if tlsRecordTypes[r.step] != ch {
				return 0, ErrBadType
			}
			r.state = tlsRecordStateVersion0
			i++
		case tlsRecordStateVersion0:
			if ch != 0x03 {
				return 0, ErrBadMajorVersion
			}
			r.state = tlsRecordStateVersion1
			i++
		case tlsRecordStateVersion1:
			if ch != tlsVersionMinors[r.step] {
				return 0, ErrBadMinorVersion
			}
			r.state = tlsRecordStateLength0
			i++
		case tlsRecordStateLength0:
			r.length = uint16(ch) << 8
			r.state = tlsRecordStateLength1
			i++
		case tlsRecordStateLength1:
			r.length |= uint16(ch)
			if r.step == 0 {
				r.length = 91
			} else if r.step == 1 {
				r.length = 1
			} else if r.length > maxTLSDataLen {
				return 0, ErrMaxDataLen
			}
			if r.length > 0 {
				r.state = tlsRecordStateData
			} else {
				r.state = tlsRecordStateType
				r.step++
			}
			i++
		case tlsRecordStateData:
			left := uint16(length - i)
			if left > r.length {
				left = r.length
			}
			if r.step >= 2 {
				skip := i - last
				copy(b[last:], b[i:length])
				length -= int(skip)
				last += int(left)
				i = last
			} else {
				i += int(left)
			}
			r.length -= left
			if r.length == 0 {
				if r.step < 3 {
					r.step++
				}
				r.state = tlsRecordStateType
			}
		}
	}

	if last == 0 {
		return 0, nil
	} else if last < length {
		length -= last
	}

	return length, nil
}

type conn struct {
	net.Conn
	rbuf           bytes.Buffer
	wbuf           bytes.Buffer
	host           string
	handshaked     chan struct{}
	parser         *obfsTLSParser
	handshakeMutex sync.Mutex
}

// newConn creates a connection for obfs-tls server.
func newConn(c net.Conn, host string) net.Conn {
	return &conn{
		Conn:       c,
		host:       host,
		handshaked: make(chan struct{}),
	}
}

func (c *conn) Handshaked() bool {
	select {
	case <-c.handshaked:
		return true
	default:
		return false
	}
}

func (c *conn) Handshake(payload []byte) (err error) {
	c.handshakeMutex.Lock()
	defer c.handshakeMutex.Unlock()

	if c.Handshaked() {
		return
	}

	if err = c.handshake(); err != nil {
		return
	}

	close(c.handshaked)
	return nil
}

func (c *conn) handshake() error {
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

func (c *conn) Read(b []byte) (n int, err error) {
	if err = c.Handshake(nil); err != nil {
		return
	}

	select {
	case <-c.handshaked:
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

func (c *conn) Write(b []byte) (n int, err error) {
	n = len(b)
	if !c.Handshaked() {
		if err = c.Handshake(b); err != nil {
			return
		}
	}

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
