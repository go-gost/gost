import socket
import struct
import sys


UDP_ECHO_HOST = "udp-echo"
UDP_ECHO_PORT = 5679


def encode_socks5_addr(host, port):
    """Encode (host, port) as SOCKS5 ADDRESS + PORT.

    Uses ATYP=3 (domain name) so Docker DNS resolves the host.
    Returns (atyp, address_bytes, port_bytes).
    """
    host_bytes = host.encode()
    addr_bytes = struct.pack("!B", len(host_bytes)) + host_bytes
    port_bytes = struct.pack("!H", port)
    return 0x03, addr_bytes, port_bytes


def build_udp_frame(payload, host, port):
    """Build a SOCKS5 UDP relay frame over TCP.

    gost HTTP UDP relay uses RSV=data-length and FRAG=0xff.
    Frame: [RSV:2][FRAG:1][ATYP:1][DST.ADDR][DST.PORT:2][DATA]
    """
    atyp, addr_bytes, port_bytes = encode_socks5_addr(host, port)
    rsv = struct.pack("!H", len(payload))
    frag = b"\xff"
    atyp_byte = struct.pack("!B", atyp)
    return rsv + frag + atyp_byte + addr_bytes + port_bytes + payload


def recvn(sock, n):
    """Receive exactly n bytes from socket."""
    buf = b""
    while len(buf) < n:
        chunk = sock.recv(n - len(buf))
        if not chunk:
            raise ConnectionError("connection closed while reading")
        buf += chunk
    return buf


def read_socks5_frame(sock):
    """Read one SOCKS5 UDP relay frame from the TCP connection.

    Returns (addr, port, data).
    """
    # Read header: RSV(2) + FRAG(1) + ATYP(1)
    header = recvn(sock, 4)
    rsv = struct.unpack("!H", header[:2])[0]
    frag = header[2]
    atyp = header[3]

    # Read address based on ATYP
    if atyp == 1:  # IPv4
        addr_bytes = recvn(sock, 4)
        addr = socket.inet_ntoa(addr_bytes)
    elif atyp == 3:  # Domain name
        domain_len = recvn(sock, 1)
        addr = recvn(sock, domain_len[0]).decode()
    elif atyp == 4:  # IPv6
        addr_bytes = recvn(sock, 16)
        addr = socket.inet_ntop(socket.AF_INET6, addr_bytes)
    else:
        raise ValueError(f"unknown ATYP: {atyp}")

    port_bytes = recvn(sock, 2)
    port = struct.unpack("!H", port_bytes)[0]

    # Read data
    if rsv > 0:
        data = recvn(sock, rsv)
    else:
        # Standard SOCKS5 UDP: read remaining
        data = b""
        while True:
            chunk = sock.recv(4096)
            if not chunk:
                break
            data += chunk

    return addr, port, data


def main():
    host = sys.argv[1] if len(sys.argv) > 1 else "127.0.0.1"
    port = int(sys.argv[2]) if len(sys.argv) > 2 else 8080

    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.settimeout(10)
    s.connect((host, port))

    # Send CONNECT with X-Gost-Protocol: udp
    req = (
        b"CONNECT 0.0.0.0:0 HTTP/1.1\r\n"
        b"Host: 0.0.0.0:0\r\n"
        b"X-Gost-Protocol: udp\r\n"
        b"Proxy-Connection: keep-alive\r\n"
        b"\r\n"
    )
    s.sendall(req)

    # Read 200 OK
    resp = b""
    while b"\r\n\r\n" not in resp:
        chunk = s.recv(4096)
        if not chunk:
            break
        resp += chunk

    if b"200" not in resp:
        print(f"FAIL: expected 200, got {resp.decode(errors='replace')}")
        sys.exit(1)

    # Build and send UDP relay frame
    payload = b"hello-gost"
    frame = build_udp_frame(payload, UDP_ECHO_HOST, UDP_ECHO_PORT)
    s.sendall(frame)

    # Read response frame
    addr, rport, data = read_socks5_frame(s)

    if b"hello-gost" in data:
        print(f"PASS: received expected data from {addr}:{rport}")
        sys.exit(0)
    else:
        print(f"FAIL: expected hello-gost in response, got {data!r}")
        sys.exit(1)


if __name__ == "__main__":
    main()
