"""DNS query client over TLS for e2e tests.

Connects to the DNS-over-TLS endpoint and performs a DNS query.

Usage:
  python3 dns_tls_query.py host port qname qtype [expected_ip]
"""
import struct
import socket
import ssl
import sys


def encode_name(name):
    parts = name.rstrip(".").split(".")
    return b"".join(bytes([len(p)]) + p.encode() for p in parts) + b"\x00"


def build_query(tid, qname, qtype):
    flags = 0x0100
    qdcount = 1
    header = struct.pack(">HHHHHH", tid, flags, qdcount, 0, 0, 0)
    qname_enc = encode_name(qname)
    question = qname_enc + struct.pack(">HH", qtype, 1)
    return header + question


def parse_response(data):
    if len(data) < 12:
        return [], -1
    flags = struct.unpack(">H", data[2:4])[0]
    rcode = flags & 0x0F
    qdcount = struct.unpack(">H", data[4:6])[0]
    ancount = struct.unpack(">H", data[6:8])[0]

    pos = 12
    for _ in range(qdcount):
        while data[pos] != 0:
            if data[pos] & 0xC0:
                pos += 2
                break
            pos += data[pos] + 1
        else:
            pos += 1
        pos += 4

    answers = []
    for _ in range(ancount):
        if data[pos] & 0xC0:
            pos += 2
        else:
            while data[pos] != 0:
                pos += data[pos] + 1
            pos += 1
        rtype, rclass, ttl, rdlength = struct.unpack(">HHIH", data[pos:pos + 10])
        pos += 10
        rdata = data[pos:pos + rdlength]
        pos += rdlength
        answers.append((rtype, rdata))

    return answers, rcode


def main():
    if len(sys.argv) < 5:
        print(f"Usage: {sys.argv[0]} host port qname qtype [expected_ip]")
        sys.exit(1)

    host = sys.argv[1]
    port = int(sys.argv[2])
    qname = sys.argv[3]
    qtype_str = sys.argv[4]
    expected = sys.argv[5] if len(sys.argv) > 5 else None

    qtype_map = {"A": 1, "AAAA": 28}
    qtype = qtype_map.get(qtype_str, 1)

    query = build_query(0x1234, qname, qtype)

    ctx = ssl.create_default_context()
    ctx.check_hostname = False
    ctx.verify_mode = ssl.CERT_NONE

    try:
        with socket.create_connection((host, port), timeout=5) as sock:
            with ctx.wrap_socket(sock, server_hostname=host) as tls_sock:
                # TCP DNS: 2-byte length prefix
                tls_sock.sendall(struct.pack(">H", len(query)) + query)
                raw = tls_sock.recv(2)
                if len(raw) < 2:
                    print("ERROR: short response header")
                    sys.exit(1)
                msglen = struct.unpack(">H", raw)[0]
                data = b""
                while len(data) < msglen:
                    chunk = tls_sock.recv(msglen - len(data))
                    if not chunk:
                        break
                    data += chunk
    except Exception as e:
        print(f"ERROR: {e}")
        sys.exit(1)

    if not data:
        print("ERROR: empty response")
        sys.exit(1)

    answers, rcode = parse_response(data)
    print(f"Got {len(answers)} answer(s), rcode={rcode}")

    for rtype, rdata in answers:
        if rtype == 1 and len(rdata) == 4:
            ip = socket.inet_ntoa(rdata)
            print(f"  A {ip}")
        elif rtype == 28 and len(rdata) == 16:
            ip6 = socket.inet_ntop(socket.AF_INET6, rdata)
            print(f"  AAAA {ip6}")
        else:
            print(f"  TYPE={rtype} RDATA={rdata.hex()}")

    if expected:
        for rtype, rdata in answers:
            if rtype == 1 and len(rdata) == 4:
                ip = socket.inet_ntoa(rdata)
                if ip == expected:
                    print(f"MATCH: expected {expected}")
                    sys.exit(0)
            elif rtype == 28 and len(rdata) == 16:
                ip6 = socket.inet_ntop(socket.AF_INET6, rdata)
                if ip6 == expected:
                    print(f"MATCH: expected {expected}")
                    sys.exit(0)
        print(f"NO MATCH: expected {expected}")
        sys.exit(1)

    sys.exit(0)


if __name__ == "__main__":
    main()
