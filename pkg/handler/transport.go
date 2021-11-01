package handler

import (
	"io"
	"sync"
)

const (
	poolBufferSize = 32 * 1024
)

var (
	pool = sync.Pool{
		New: func() interface{} {
			return make([]byte, poolBufferSize)
		},
	}
)

func Transport(rw1, rw2 io.ReadWriter) error {
	errc := make(chan error, 1)
	go func() {
		errc <- copyBuffer(rw1, rw2)
	}()

	go func() {
		errc <- copyBuffer(rw2, rw1)
	}()

	err := <-errc
	if err != nil && err == io.EOF {
		err = nil
	}
	return err
}

func copyBuffer(dst io.Writer, src io.Reader) error {
	buf := pool.Get().([]byte)
	defer pool.Put(buf)

	_, err := io.CopyBuffer(dst, src, buf)
	return err
}
