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

type sshConn struct {
	channel ssh.Channel
	net.Conn
}

func NewConn(conn net.Conn, channel ssh.Channel) net.Conn {
	return &sshConn{
		Conn:    conn,
		channel: channel,
	}
}

func (c *sshConn) Read(b []byte) (n int, err error) {
	return c.channel.Read(b)
}

func (c *sshConn) Write(b []byte) (n int, err error) {
	return c.channel.Write(b)
}

func (c *sshConn) Close() error {
	return c.channel.Close()
}
