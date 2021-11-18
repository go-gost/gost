package relay

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"net"
)

type packetConn struct {
	net.Conn
}

func UDPTunConn(conn net.Conn) net.Conn {
	return &packetConn{
		Conn: conn,
	}
}

func (c *packetConn) Read(b []byte) (n int, err error) {
	var bb [2]byte
	_, err = io.ReadFull(c.Conn, bb[:])
	if err != nil {
		return
	}

	dlen := int(binary.BigEndian.Uint16(bb[:]))
	if len(b) >= dlen {
		return io.ReadFull(c.Conn, b[:dlen])
	}
	buf := make([]byte, dlen)
	_, err = io.ReadFull(c.Conn, buf)
	n = copy(b, buf)

	return
}

func (c *packetConn) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	n, err = c.Read(b)
	addr = c.Conn.RemoteAddr()
	return
}

func (c *packetConn) Write(b []byte) (n int, err error) {
	if len(b) > math.MaxUint16 {
		err = errors.New("write: data maximum exceeded")
		return
	}

	var bb [2]byte
	binary.BigEndian.PutUint16(bb[:2], uint16(len(b)))
	_, err = c.Conn.Write(bb[:])
	if err != nil {
		return
	}
	return c.Conn.Write(b)
}
