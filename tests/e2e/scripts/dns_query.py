"""DNS query client for e2e tests.

Usage:
  python3 dns_query.py udp host port qname qtype [expected_ip|empty]
  python3 dns_query.py tcp host port qname qtype [expected_ip|empty]

Sends a DNS query and optionally checks the response:
  - expected_ip: verify response contains this IP address
  - "empty":     verify response has zero answer records (NXDOMAIN or blocked)
  - omitted:     exit 0 on any valid DNS response

Exits 0 on success, 1 on failure.
"""
import struct
import socket
import sys


def encode_name(name):
    parts = name.rstrip(".").split(".")
    return b"".join(bytes([len(p)]) + p.encode() for p in parts) + b"\x00"


def skip_name(data, pos):
    """Skip one DNS name at pos, return position after it."""
    while True:
        length = data[pos]
        if length == 0:
            return pos + 1
        if length & 0xC0:
            return pos + 2
        pos += length + 1


def build_query(qname, qtype):
    tid = 0x1234
    flags = 0x0100  # RD=1
    qdcount = 1
    header = struct.pack(">HHHHHH", tid, flags, qdcount, 0, 0, 0)
    qname_enc = encode_name(qname)
    question = qname_enc + struct.pack(">HH", qtype, 1)  # QTYPE, QCLASS=IN
    return header + question


def parse_response(data):
    if len(data) < 12:
        return [], -1
    flags = struct.unpack(">H", data[2:4])[0]
    rcode = flags & 0x0F
    qdcount = struct.unpack(">H", data[4:6])[0]
    ancount = struct.unpack(">H", data[6:8])[0]

    # Skip question section
    pos = 12
    for _ in range(qdcount):
        pos = skip_name(data, pos) + 4  # QTYPE + QCLASS

    answers = []
    for _ in range(ancount):
        pos = skip_name(data, pos)
        rtype, rclass, ttl, rdlength = struct.unpack(">HHIH", data[pos:pos + 10])
        pos += 10
        rdata = data[pos:pos + rdlength]
        pos += rdlength
        answers.append((rtype, rclass, ttl, rdata))

    return answers, rcode


def query_udp(host, port, query):
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    sock.settimeout(5)
    sock.sendto(query, (host, port))
    data, _ = sock.recvfrom(4096)
    sock.close()
    return data


def query_tcp(host, port, query):
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.settimeout(5)
    sock.connect((host, port))
    sock.sendall(struct.pack(">H", len(query)) + query)
    raw = sock.recv(2)
    if len(raw) < 2:
        sock.close()
        return b""
    msglen = struct.unpack(">H", raw)[0]
    data = b""
    while len(data) < msglen:
        chunk = sock.recv(msglen - len(data))
        if not chunk:
            break
        data += chunk
    sock.close()
    return data


def format_ip(rtype, rdata):
    if rtype == 1 and len(rdata) == 4:
        return socket.inet_ntoa(rdata)
    if rtype == 28 and len(rdata) == 16:
        return socket.inet_ntop(socket.AF_INET6, rdata)
    return repr(rdata)


def main():
    if len(sys.argv) < 5:
        print("Usage: dns_query.py <udp|tcp> <host> <port> <qname> <qtype> [expected_ip|empty]")
        sys.exit(1)

    mode = sys.argv[1]
    host = sys.argv[2]
    port = int(sys.argv[3])
    qname = sys.argv[4]
    qtype_str = sys.argv[5]
    expected = sys.argv[6] if len(sys.argv) > 6 else None

    qtype_map = {"A": 1, "AAAA": 28}
    qtype = qtype_map.get(qtype_str, 1)

    query = build_query(qname, qtype)

    try:
        if mode == "tcp":
            data = query_tcp(host, port, query)
        else:
            data = query_udp(host, port, query)
    except Exception as e:
        print(f"ERROR: {e}")
        sys.exit(1)

    if not data:
        print("ERROR: empty response")
        sys.exit(1)

    answers, rcode = parse_response(data)
    print(f"Got {len(answers)} answer(s), rcode={rcode}")

    for rtype, _, _, rdata in answers:
        print(f"  {format_ip(rtype, rdata)}")

    if expected == "empty":
        if len(answers) == 0:
            print("MATCH: empty answer (expected)")
            sys.exit(0)
        print(f"NO MATCH: expected empty, got {len(answers)} answers")
        sys.exit(1)
    elif expected:
        for rtype, _, _, rdata in answers:
            ip = format_ip(rtype, rdata)
            if ip == expected:
                print(f"MATCH: expected {expected}")
                sys.exit(0)
        print(f"NO MATCH: expected {expected}")
        sys.exit(1)
    else:
        sys.exit(0)


if __name__ == "__main__":
    main()
