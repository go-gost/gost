package relay

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/go-gost/gost/pkg/logger"
	"github.com/go-gost/relay"
)

type conn struct {
	net.Conn
	udp        bool
	wbuf       bytes.Buffer
	once       sync.Once
	headerSent bool
	logger     logger.Logger
}

func (c *conn) Read(b []byte) (n int, err error) {
	c.once.Do(func() {
		resp := relay.Response{}
		_, err = resp.ReadFrom(c.Conn)
		if err != nil {
			return
		}
		if resp.Version != relay.Version1 {
			err = relay.ErrBadVersion
			return
		}
		if resp.Status != relay.StatusOK {
			err = fmt.Errorf("status %d", resp.Status)
			return
		}
	})

	if err != nil {
		return
	}

	if !c.udp {
		return c.Conn.Read(b)
	}

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

func (c *conn) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	n, err = c.Read(b)
	addr = c.Conn.RemoteAddr()
	return
}

func (c *conn) Write(b []byte) (n int, err error) {
	if len(b) > 0xFFFF {
		err = errors.New("write: data maximum exceeded")
		return
	}
	n = len(b) // force byte length consistent
	if c.wbuf.Len() > 0 {
		if c.udp {
			var bb [2]byte
			binary.BigEndian.PutUint16(bb[:2], uint16(len(b)))
			c.wbuf.Write(bb[:])
			c.headerSent = true
		}
		c.wbuf.Write(b) // append the data to the cached header
		// _, err = c.Conn.Write(c.wbuf.Bytes())
		// c.wbuf.Reset()
		_, err = c.wbuf.WriteTo(c.Conn)
		return
	}

	if !c.udp {
		return c.Conn.Write(b)
	}
	if !c.headerSent {
		c.headerSent = true
		b2 := make([]byte, len(b)+2)
		copy(b2, b)
		_, err = c.Conn.Write(b2)
		return
	}
	nsize := 2 + len(b)
	var buf []byte
	if nsize <= mediumBufferSize {
		buf = mPool.Get().([]byte)
		defer mPool.Put(buf)
	} else {
		buf = make([]byte, nsize)
	}
	binary.BigEndian.PutUint16(buf[:2], uint16(len(b)))
	n = copy(buf[2:], b)
	_, err = c.Conn.Write(buf[:nsize])
	return
}

func (c *relayConn) WriteTo(b []byte, addr net.Addr) (n int, err error) {
	return c.Write(b)
}
