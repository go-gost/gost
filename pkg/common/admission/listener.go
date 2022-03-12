package admission

import (
	"net"

	"github.com/go-gost/gost/pkg/admission"
)

type listener struct {
	net.Listener
	admission admission.Admission
}

func WrapListener(admission admission.Admission, ln net.Listener) net.Listener {
	if admission == nil {
		return ln
	}
	return &listener{
		Listener:  ln,
		admission: admission,
	}
}

func (ln *listener) Accept() (net.Conn, error) {
	for {
		c, err := ln.Listener.Accept()
		if err != nil {
			return nil, err
		}
		if ln.admission != nil &&
			!ln.admission.Admit(c.RemoteAddr().String()) {
			c.Close()
			continue
		}
		return c, err
	}
}
