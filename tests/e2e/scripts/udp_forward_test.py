import socket
import sys


def main():
    host = sys.argv[1] if len(sys.argv) > 1 else "127.0.0.1"
    port = int(sys.argv[2]) if len(sys.argv) > 2 else 9000

    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    sock.settimeout(5)

    payload = b"hello-gost-udp"
    sock.sendto(payload, (host, port))

    data, addr = sock.recvfrom(2048)
    if data == payload:
        print("PASS: received echo")
        sys.exit(0)
    else:
        print(f"FAIL: expected {payload!r}, got {data!r}")
        sys.exit(1)


if __name__ == "__main__":
    main()
