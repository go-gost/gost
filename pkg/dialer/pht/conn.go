package pht

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/go-gost/gost/pkg/logger"
)

type conn struct {
	cid        string
	addr       string
	client     *http.Client
	tlsEnabled bool
	buf        []byte
	rxc        chan []byte
	closed     chan struct{}
	md         metadata
	logger     logger.Logger
}

func (c *conn) Read(b []byte) (n int, err error) {
	if len(c.buf) == 0 {
		select {
		case c.buf = <-c.rxc:
		case <-c.closed:
			err = net.ErrClosed
			return
		}
	}

	n = copy(b, c.buf)
	c.buf = c.buf[n:]

	return
}

func (c *conn) Write(b []byte) (n int, err error) {
	if len(b) == 0 {
		return
	}

	buf := bytes.NewBufferString(base64.StdEncoding.EncodeToString(b))
	buf.WriteByte('\n')

	var url string
	if c.tlsEnabled {
		url = fmt.Sprintf("https://%s%s?token=%s", c.addr, c.md.pushPath, c.cid)
	} else {
		url = fmt.Sprintf("http://%s%s?token=%s", c.addr, c.md.pushPath, c.cid)
	}
	r, err := http.NewRequest(http.MethodPost, url, buf)
	if err != nil {
		return
	}

	resp, err := c.client.Do(r)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = errors.New(resp.Status)
		return
	}

	n = len(b)
	return
}

func (c *conn) readLoop() {
	defer c.Close()

	var url string
	if c.tlsEnabled {
		url = fmt.Sprintf("https://%s%s?token=%s", c.addr, c.md.pullPath, c.cid)
	} else {
		url = fmt.Sprintf("http://%s%s?token=%s", c.addr, c.md.pullPath, c.cid)
	}
	for {
		err := func() error {
			r, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				return err
			}

			resp, err := c.client.Do(r)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return errors.New(resp.Status)
			}

			scanner := bufio.NewScanner(resp.Body)
			for scanner.Scan() {
				b, err := base64.StdEncoding.DecodeString(scanner.Text())
				if err != nil {
					return err
				}
				select {
				case c.rxc <- b:
				case <-c.closed:
					return net.ErrClosed
				}
			}

			return scanner.Err()
		}()

		if err != nil {
			c.logger.Error(err)
			return
		}
	}
}

func (c *conn) LocalAddr() net.Addr {
	return &net.TCPAddr{}
}

func (c *conn) RemoteAddr() net.Addr {
	addr, _ := net.ResolveTCPAddr("tcp", c.addr)
	if addr == nil {
		addr = &net.TCPAddr{}
	}

	return addr
}

func (c *conn) Close() error {
	select {
	case <-c.closed:
	default:
		close(c.closed)
	}
	return nil
}

func (c *conn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *conn) SetWriteDeadline(t time.Time) error {
	return nil
}

func (c *conn) SetDeadline(t time.Time) error {
	return nil
}
