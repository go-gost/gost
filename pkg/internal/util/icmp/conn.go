package icmp

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net"
	"sync/atomic"

	"github.com/go-gost/gost/pkg/common/bufpool"
	"github.com/go-gost/gost/pkg/logger"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

const (
	readBufferSize  = 1500
	writeBufferSize = 1500
	magicNumber     = 0x474F5354
)

const (
	messageHeaderLen = 10
)

const (
	FlagAck = 1
)

var (
	ErrInvalidPacket = errors.New("icmp: invalid packet")
	ErrInvalidType   = errors.New("icmp: invalid type")
	ErrShortBuffer   = errors.New("icmp: short buffer")
)

type message struct {
	// magic uint32 // magic number
	flags uint16 // flags
	// rsv   uint16 // reserved field
	// len   uint16 // length of data
	data []byte
}

func (m *message) Encode(b []byte) (n int, err error) {
	if len(b) < messageHeaderLen+len(m.data) {
		err = ErrShortBuffer
		return
	}
	binary.BigEndian.PutUint32(b[:4], magicNumber) // magic number
	binary.BigEndian.PutUint16(b[4:6], m.flags)    // flags
	binary.BigEndian.PutUint16(b[6:8], 0)          // reserved
	binary.BigEndian.PutUint16(b[8:10], uint16(len(m.data)))
	copy(b[messageHeaderLen:], m.data)

	n = messageHeaderLen + len(m.data)
	return
}

func (m *message) Decode(b []byte) (n int, err error) {
	if len(b) < messageHeaderLen {
		err = ErrShortBuffer
		return
	}
	if binary.BigEndian.Uint32(b[:4]) != magicNumber {
		err = ErrInvalidPacket
		return
	}
	m.flags = binary.BigEndian.Uint16(b[4:6])
	length := binary.BigEndian.Uint16(b[8:10])
	if len(b[messageHeaderLen:]) < int(length) {
		err = ErrShortBuffer
		return
	}
	m.data = b[messageHeaderLen : messageHeaderLen+length]

	n = messageHeaderLen + int(length)
	return
}

type clientConn struct {
	net.PacketConn
	id  int
	seq uint32
}

func ClientConn(conn net.PacketConn, id int) net.PacketConn {
	return &clientConn{
		PacketConn: conn,
		id:         id,
	}
}

func (c *clientConn) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	buf := bufpool.Get(readBufferSize)
	defer bufpool.Put(buf)

	for {
		n, addr, err = c.PacketConn.ReadFrom(*buf)
		if err != nil {
			return
		}

		m, err := icmp.ParseMessage(1, (*buf)[:n])
		if err != nil {
			// logger.Default().Error("icmp: parse message %v", err)
			return 0, addr, err
		}
		echo, ok := m.Body.(*icmp.Echo)
		if !ok || m.Type != ipv4.ICMPTypeEchoReply {
			// logger.Default().Warnf("icmp: invalid type %s (discarded)", m.Type)
			continue // discard
		}

		if echo.ID != c.id {
			// logger.Default().Warnf("icmp: id mismatch got %d, should be %d (discarded)", echo.ID, c.id)
			continue
		}

		msg := message{}
		if _, err := msg.Decode(echo.Data); err != nil {
			logger.Default().Warn(err)
			continue
		}

		if msg.flags&FlagAck == 0 {
			// logger.Default().Warn("icmp: invalid message (discarded)")
			continue
		}
		n = copy(b, msg.data)
		break
	}

	if v, ok := addr.(*net.IPAddr); ok {
		addr = &net.UDPAddr{
			IP:   v.IP,
			Port: c.id,
		}
	}
	// logger.Default().Infof("icmp: read from: %v %d", addr, n)

	return
}

func (c *clientConn) WriteTo(b []byte, addr net.Addr) (n int, err error) {
	// logger.Default().Infof("icmp: write to: %v %d", addr, len(b))
	switch v := addr.(type) {
	case *net.UDPAddr:
		addr = &net.IPAddr{IP: v.IP}
	}

	buf := bufpool.Get(writeBufferSize)
	defer bufpool.Put(buf)

	msg := message{
		data: b,
	}
	nn, err := msg.Encode(*buf)
	if err != nil {
		return
	}

	echo := icmp.Echo{
		ID:   c.id,
		Seq:  int(atomic.AddUint32(&c.seq, 1)),
		Data: (*buf)[:nn],
	}
	m := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &echo,
	}
	wb, err := m.Marshal(nil)
	if err != nil {
		return 0, err
	}
	_, err = c.PacketConn.WriteTo(wb, addr)
	n = len(b)
	return
}

type serverConn struct {
	net.PacketConn
	seqs [65535]uint32
}

func ServerConn(conn net.PacketConn) net.PacketConn {
	return &serverConn{
		PacketConn: conn,
	}
}

func (c *serverConn) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	buf := bufpool.Get(readBufferSize)
	defer bufpool.Put(buf)

	for {
		n, addr, err = c.PacketConn.ReadFrom(*buf)
		if err != nil {
			return
		}

		m, err := icmp.ParseMessage(1, (*buf)[:n])
		if err != nil {
			// logger.Default().Error("icmp: parse message %v", err)
			return 0, addr, err
		}

		echo, ok := m.Body.(*icmp.Echo)
		if !ok || m.Type != ipv4.ICMPTypeEcho || echo.ID <= 0 {
			// logger.Default().Warnf("icmp: invalid type %s (discarded)", m.Type)
			continue
		}

		atomic.StoreUint32(&c.seqs[uint16(echo.ID-1)], uint32(echo.Seq))

		msg := message{}
		if _, err := msg.Decode(echo.Data); err != nil {
			continue
		}

		if msg.flags&FlagAck > 0 {
			continue
		}

		n = copy(b, msg.data)

		if v, ok := addr.(*net.IPAddr); ok {
			addr = &net.UDPAddr{
				IP:   v.IP,
				Port: echo.ID,
			}
		}
		break
	}

	// logger.Default().Infof("icmp: read from: %v %d", addr, n)

	return
}

func (c *serverConn) WriteTo(b []byte, addr net.Addr) (n int, err error) {
	// logger.Default().Infof("icmp: write to: %v %d", addr, len(b))
	var id int
	switch v := addr.(type) {
	case *net.UDPAddr:
		addr = &net.IPAddr{IP: v.IP}
		id = v.Port
	}

	if id <= 0 || id > math.MaxUint16 {
		err = fmt.Errorf("icmp: invalid message id %v", addr)
		return
	}

	buf := bufpool.Get(writeBufferSize)
	defer bufpool.Put(buf)

	msg := message{
		flags: FlagAck,
		data:  b,
	}
	nn, err := msg.Encode(*buf)
	if err != nil {
		return
	}

	echo := icmp.Echo{
		ID:   id,
		Seq:  int(atomic.LoadUint32(&c.seqs[id-1])),
		Data: (*buf)[:nn],
	}
	m := icmp.Message{
		Type: ipv4.ICMPTypeEchoReply,
		Code: 0,
		Body: &echo,
	}
	wb, err := m.Marshal(nil)
	if err != nil {
		return 0, err
	}
	_, err = c.PacketConn.WriteTo(wb, addr)
	n = len(b)
	return
}
