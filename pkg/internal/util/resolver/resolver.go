package resolver

import (
	"net"

	"github.com/miekg/dns"
)

func AddSubnetOpt(m *dns.Msg, ip net.IP) {
	if m == nil || ip == nil {
		return
	}

	opt := new(dns.OPT)
	opt.Hdr.Name = "."
	opt.Hdr.Rrtype = dns.TypeOPT
	e := new(dns.EDNS0_SUBNET)
	e.Code = dns.EDNS0SUBNET
	if ip := ip.To4(); ip != nil {
		e.Family = 1
		e.SourceNetmask = 24
		e.Address = ip
	} else {
		e.Family = 2
		e.SourceNetmask = 128
		e.Address = ip.To16()
	}
	opt.Option = append(opt.Option, e)
	m.Extra = append(m.Extra, opt)
}
