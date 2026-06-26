"""Simple authoritative DNS responder for TCP mode e2e tests.

Listens on TCP port 5353 and responds with the same static records
as dns_responder.py, but over TCP (RFC 5966).
"""
import socketserver
import struct
import socket


def decode_name(data, offset):
    labels = []
    while True:
        length = data[offset]
        if length == 0:
            offset += 1
            break
        if length & 0xC0:
            offset += 2
            break
        offset += 1
        labels.append(data[offset:offset + length].decode())
        offset += length
    return '.'.join(labels), offset


RECORDS = {
    ("test.example.com", 1): (1, 300, socket.inet_aton("10.0.0.1")),
    ("test2.example.com", 1): (1, 300, socket.inet_aton("10.0.0.2")),
    ("example.com", 28): (28, 300, socket.inet_pton(socket.AF_INET6, "::1")),
}


class DNSResponder(socketserver.StreamRequestHandler):
    def handle(self):
        # TCP DNS: 2-byte length prefix
        raw = self.rfile.read(2)
        if len(raw) < 2:
            return
        msglen = struct.unpack(">H", raw)[0]
        data = self.rfile.read(msglen)
        if len(data) < 12:
            return

        tid = struct.unpack(">H", data[:2])[0]
        qdcount = struct.unpack(">H", data[4:6])[0]

        if qdcount == 0:
            return

        qname, pos = decode_name(data, 12)
        qtype = struct.unpack(">H", data[pos:pos + 2])[0]
        qclass = struct.unpack(">H", data[pos + 2:pos + 4])[0]

        key = (qname, qtype)
        if key in RECORDS:
            rcode = 0
            ancount = 1
            rtype, ttl, rdata = RECORDS[key]
        else:
            rcode = 3  # NXDOMAIN
            ancount = 0
            rtype, ttl, rdata = 0, 0, b""

        flags = 0x8000 | 0x0400 | rcode
        header = struct.pack(">HHHHHH", tid, flags, qdcount, ancount, 0, 0)
        question = data[12:pos + 4]

        answer = b""
        if ancount:
            answer = (
                struct.pack(">HH", 0xC00C, rtype)
                + struct.pack(">HI", qclass, ttl)
                + struct.pack(">H", len(rdata)) + rdata
            )

        dnsmsg = header + question + answer
        self.wfile.write(struct.pack(">H", len(dnsmsg)) + dnsmsg)


if __name__ == "__main__":
    with socketserver.TCPServer(("0.0.0.0", 5353), DNSResponder) as srv:
        srv.serve_forever()
