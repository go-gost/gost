"""DNS responder for resolver e2e tests.

Returns the target IP (argv[1]) for an A query of "echo.test" and
NXDOMAIN for every other query. This lets a gost proxy configured with
this server as its resolver resolve a made-up hostname to a reachable
echo server, proving the resolver drives outbound dialing.

Listens on UDP port argv[2] (default 5353).
"""
import socketserver
import struct
import socket
import sys

TARGET_IP = sys.argv[1] if len(sys.argv) > 1 else "10.0.0.1"
PORT = int(sys.argv[2]) if len(sys.argv) > 2 else 5353
MATCH_NAME = "echo.test"


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

        if qname == MATCH_NAME and qtype == 1:
            rcode, ancount = 0, 1
            rtype, ttl, rdata = 1, 300, socket.inet_aton(TARGET_IP)
        else:
            rcode, ancount = 3, 0  # NXDOMAIN
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
    socketserver.ThreadingUDPServer.allow_reuse_address = True
    with socketserver.ThreadingUDPServer(("0.0.0.0", PORT), DNSResponder) as srv:
        srv.serve_forever()
