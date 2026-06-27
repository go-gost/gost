import socket
import sys
import time


def main():
    host = sys.argv[1] if len(sys.argv) > 1 else "127.0.0.1"
    port = int(sys.argv[2]) if len(sys.argv) > 2 else 8000
    idle_timeout = int(sys.argv[3]) if len(sys.argv) > 3 else 3

    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.settimeout(10)
    s.connect((host, port))

    # Send an HTTP GET request — the forward handler pipes us to tcp-echo:5678
    req = b"GET / HTTP/1.0\r\nHost: tcp-echo\r\n\r\n"
    s.sendall(req)

    # Read response — should contain "hello-gost" (echo server reply)
    resp = b""
    while b"hello-gost" not in resp:
        chunk = s.recv(4096)
        if not chunk:
            break
        resp += chunk

    if b"hello-gost" not in resp:
        print(f"FAIL: expected hello-gost in response, got {resp.decode(errors='replace')}")
        sys.exit(1)
    print(f"PASS: first request through forward pipe succeeded")

    # Wait longer than idleTimeout
    wait = idle_timeout + 2
    print(f"Waiting {wait}s for idle timeout...")
    time.sleep(wait)

    # Try to send more data — idle timeout should have closed the pipe
    try:
        s.sendall(b"GET / HTTP/1.0\r\nHost: tcp-echo\r\n\r\n")
        data = s.recv(4096)
        if not data:
            print("PASS: connection closed after idle timeout (empty recv)")
            sys.exit(0)
        # Got data means the pipe is still alive
        print(f"FAIL: connection still alive after idle timeout, got {data!r}")
        sys.exit(1)
    except (socket.timeout, ConnectionResetError, BrokenPipeError, OSError) as e:
        print(f"PASS: connection closed after idle timeout: {e}")
        sys.exit(0)


if __name__ == "__main__":
    main()
