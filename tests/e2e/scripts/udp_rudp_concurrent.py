"""Concurrent UDP sender for the gost#911 reverse-tunnel regression test.

Each socket is an independent source endpoint sharing ONE reverse tunnel
stream. Every datagram is a run of its own endpoint id, so a reply that does
not match the id byte exactly is a frame corrupted by an interleaved writer
rather than plain loss.

Usage: udp_rudp_concurrent.py <host> <port> <count> <duration_seconds>
"""

import socket
import sys
import threading
import time


def main():
    host = sys.argv[1]
    port = int(sys.argv[2])
    count = int(sys.argv[3])
    duration = float(sys.argv[4])

    payload_len = 1200
    end = time.monotonic() + duration
    results = {}
    lock = threading.Lock()

    def endpoint(endpoint_id):
        payload = bytes([endpoint_id]) * payload_len
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.bind(("0.0.0.0", 0))
        sock.settimeout(0.2)

        sent = received = damaged = 0
        largest_gap = 0.0
        last = time.monotonic()

        while time.monotonic() < end:
            try:
                sock.sendto(payload, (host, port))
                sent += 1
            except OSError:
                pass
            while True:
                try:
                    data, _ = sock.recvfrom(65535)
                except socket.timeout:
                    break
                except OSError:
                    break
                received += 1
                if data != payload:
                    damaged += 1
                now = time.monotonic()
                largest_gap = max(largest_gap, now - last)
                last = now

        largest_gap = max(largest_gap, time.monotonic() - last)
        with lock:
            results[endpoint_id] = (sent, received, damaged, round(largest_gap, 3))
        sock.close()

    threads = [
        threading.Thread(target=endpoint, args=(i + 1,)) for i in range(count)
    ]
    for t in threads:
        t.start()
    for t in threads:
        t.join()

    failures = []
    for endpoint_id in sorted(results):
        sent, received, damaged, gap = results[endpoint_id]
        loss_pct = 100.0 * (1.0 - received / sent) if sent else 100.0
        print(
            f"endpoint {endpoint_id}: sent={sent} received={received} "
            f"damaged={damaged} loss={loss_pct:.1f}% max_gap={gap}"
        )
        # Unsent datagrams are outside our control (the UDP socket's own
        # buffer); anything that comes BACK must be byte-identical.
        if damaged:
            failures.append(
                f"endpoint {endpoint_id}: {damaged} datagram(s) came back "
                f"corrupted — frames interleaved on the shared tunnel stream"
            )
        # A torn frame makes the far end read io.ErrUnexpectedEOF, which
        # rebuilds the whole reverse tunnel. That shows up as a receive gap of
        # seconds, not milliseconds.
        if gap >= 1.0:
            failures.append(
                f"endpoint {endpoint_id}: {gap}s receive gap — the reverse "
                f"tunnel was torn down and rebound under active traffic"
            )

    if failures:
        for f in failures:
            print(f"FAIL: {f}", file=sys.stderr)
        sys.exit(1)

    print(f"PASS: {count} concurrent endpoints, no corruption, no tunnel reset")
    sys.exit(0)


if __name__ == "__main__":
    main()