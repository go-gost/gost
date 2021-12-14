package http2

import (
	"errors"
	"io"
	"net/http"
)

type readWriter struct {
	r io.Reader
	w io.Writer
}

func (rw *readWriter) Read(p []byte) (n int, err error) {
	return rw.r.Read(p)
}

func (rw *readWriter) Write(p []byte) (n int, err error) {
	return rw.w.Write(p)
}

type flushWriter struct {
	w io.Writer
}

func (fw flushWriter) Write(p []byte) (n int, err error) {
	defer func() {
		if r := recover(); r != nil {
			if s, ok := r.(string); ok {
				err = errors.New(s)
				return
			}
			err = r.(error)
		}
	}()

	n, err = fw.w.Write(p)
	if err != nil {
		// log.Log("flush writer:", err)
		return
	}
	if f, ok := fw.w.(http.Flusher); ok {
		f.Flush()
	}
	return
}
