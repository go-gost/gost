package grpc

import (
	"errors"
	"io"
	"net"
	"time"

	pb "github.com/go-gost/gost/pkg/common/util/grpc/proto"
)

type conn struct {
	c          pb.GostTunel_TunnelClient
	rb         []byte
	localAddr  net.Addr
	remoteAddr net.Addr
	closed     chan struct{}
}

func (c *conn) Read(b []byte) (n int, err error) {
	select {
	case <-c.c.Context().Done():
		err = c.c.Context().Err()
		return
	case <-c.closed:
		err = io.ErrClosedPipe
		return
	default:
	}

	if len(c.rb) == 0 {
		chunk, err := c.c.Recv()
		if err != nil {
			return 0, err
		}
		c.rb = chunk.Data
	}

	n = copy(b, c.rb)
	c.rb = c.rb[n:]
	return
}

func (c *conn) Write(b []byte) (n int, err error) {
	select {
	case <-c.c.Context().Done():
		err = c.c.Context().Err()
		return
	case <-c.closed:
		err = io.ErrClosedPipe
		return
	default:
	}

	if err = c.c.Send(&pb.Chunk{
		Data: b,
	}); err != nil {
		return
	}
	n = len(b)
	return
}

func (c *conn) Close() error {
	select {
	case <-c.closed:
	default:
		close(c.closed)
	}

	return nil
}

func (c *conn) LocalAddr() net.Addr {
	return c.localAddr
}

func (c *conn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *conn) SetDeadline(t time.Time) error {
	return &net.OpError{Op: "set", Net: "grpc", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}

func (c *conn) SetReadDeadline(t time.Time) error {
	return &net.OpError{Op: "set", Net: "grpc", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}

func (c *conn) SetWriteDeadline(t time.Time) error {
	return &net.OpError{Op: "set", Net: "grpc", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}
