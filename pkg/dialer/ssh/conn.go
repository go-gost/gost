package ssh

import (
	"net"

	"golang.org/x/crypto/ssh"
)

type sshSession struct {
	addr   string
	conn   net.Conn
	client *ssh.Client
	closed chan struct{}
	dead   chan struct{}
}

func (s *sshSession) IsClosed() bool {
	select {
	case <-s.dead:
		return true
	case <-s.closed:
		return true
	default:
	}
	return false
}

func (s *sshSession) wait() error {
	defer close(s.closed)
	return s.client.Wait()
}
