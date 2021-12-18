package ssh

import (
	"net"

	"golang.org/x/crypto/ssh"
)

// a dummy ssh client conn used by client connector
type ClientConn struct {
	net.Conn
	client *ssh.Client
}

func NewClientConn(conn net.Conn, client *ssh.Client) net.Conn {
	return &ClientConn{
		Conn:   conn,
		client: client,
	}
}

func (c *ClientConn) Client() *ssh.Client {
	return c.client
}
