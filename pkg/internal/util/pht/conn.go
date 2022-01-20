package pht

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"errors"
	"net"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/go-gost/gost/pkg/logger"
)

type clientConn struct {
	client     *http.Client
	pushURL    string
	pullURL    string
	buf        []byte
	rxc        chan []byte
	closed     chan struct{}
	localAddr  net.Addr
	remoteAddr net.Addr
	logger     logger.Logger
}

func (c *clientConn) Read(b []byte) (n int, err error) {
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

func (c *clientConn) Write(b []byte) (n int, err error) {
	if len(b) == 0 {
		return
	}

	buf := bytes.NewBufferString(base64.StdEncoding.EncodeToString(b))
	buf.WriteByte('\n')

	r, err := http.NewRequest(http.MethodPost, c.pushURL, buf)
	if err != nil {
		return
	}
	if c.logger.IsLevelEnabled(logger.DebugLevel) {
		dump, _ := httputil.DumpRequest(r, false)
		c.logger.Debug(string(dump))
	}

	resp, err := c.client.Do(r)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if c.logger.IsLevelEnabled(logger.DebugLevel) {
		dump, _ := httputil.DumpResponse(resp, false)
		c.logger.Debug(string(dump))
	}

	if resp.StatusCode != http.StatusOK {
		err = errors.New(resp.Status)
		return
	}

	n = len(b)
	return
}

func (c *clientConn) readLoop() {
	defer c.Close()

	for {
		err := func() error {
			r, err := http.NewRequest(http.MethodGet, c.pullURL, nil)
			if err != nil {
				return err
			}
			if c.logger.IsLevelEnabled(logger.DebugLevel) {
				dump, _ := httputil.DumpRequest(r, false)
				c.logger.Debug(string(dump))
			}

			resp, err := c.client.Do(r)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if c.logger.IsLevelEnabled(logger.DebugLevel) {
				dump, _ := httputil.DumpResponse(resp, false)
				c.logger.Debug(string(dump))
			}

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

func (c *clientConn) LocalAddr() net.Addr {
	return c.localAddr
}

func (c *clientConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *clientConn) Close() error {
	select {
	case <-c.closed:
	default:
		close(c.closed)
	}
	return nil
}

func (c *clientConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *clientConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func (c *clientConn) SetDeadline(t time.Time) error {
	return nil
}
