package quic

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
	"net"
)

type cipherConn struct {
	net.PacketConn
	key []byte
}

func CipherPacketConn(conn net.PacketConn, key []byte) net.PacketConn {
	return &cipherConn{
		PacketConn: conn,
		key:        key,
	}
}

func (conn *cipherConn) ReadFrom(data []byte) (n int, addr net.Addr, err error) {
	n, addr, err = conn.PacketConn.ReadFrom(data)
	if err != nil {
		return
	}
	b, err := conn.decrypt(data[:n])
	if err != nil {
		return
	}

	copy(data, b)

	return len(b), addr, nil
}

func (conn *cipherConn) WriteTo(data []byte, addr net.Addr) (n int, err error) {
	b, err := conn.encrypt(data)
	if err != nil {
		return
	}

	_, err = conn.PacketConn.WriteTo(b, addr)
	if err != nil {
		return
	}

	return len(b), nil
}

func (conn *cipherConn) encrypt(data []byte) ([]byte, error) {
	c, err := aes.NewCipher(conn.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, data, nil), nil
}

func (conn *cipherConn) decrypt(data []byte) ([]byte, error) {
	c, err := aes.NewCipher(conn.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
