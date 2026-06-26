"""Simple authoritative DNS responder for e2e tests.

Listens on UDP port 5353 and responds with static records:

  test.example.com.   IN A     10.0.0.1
  test2.example.com.  IN A     10.0.0.2
  example.com.        IN AAAA  ::1

All other queries receive NXDOMAIN.
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


def encode_name(name):
    parts = name.rstrip(".").split(".")
    return b"".join(bytes([len(p)]) + p.encode() for p in parts) + b"\x00"


RECORDS = {
    ("test.example.com", 1): (1, 300, socket.inet_aton("10.0.0.1")),
    ("test2.example.com", 1): (1, 300, socket.inet_aton("10.0.0.2")),
    ("example.com", 28): (28, 300, socket.inet_pton(socket.AF_INET6, "::1")),
}


class DNSResponder(socketserver.DatagramRequestHandler):
    def handle(self):
        data = self.rfile.read(512)
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

        flags = 0x8000 | 0x0400 | rcode  # QR=1, AA=1, rcode
        header = struct.pack(">HHHHHH", tid, flags, qdcount, ancount, 0, 0)

        question = data[12:pos + 4]

        answer = b""
        if ancount:
            answer = (
                struct.pack(">HH", 0xC00C, rtype)  # NAME pointer + TYPE
                + struct.pack(">HI", qclass, ttl)   # CLASS + TTL
                + struct.pack(">H", len(rdata)) + rdata  # RDLENGTH + RDATA
            )

        self.wfile.write(header + question + answer)


if __name__ == "__main__":
    with socketserver.UDPServer(("0.0.0.0", 5353), DNSResponder) as srv:
        srv.serve_forever()
