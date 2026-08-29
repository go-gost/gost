# Raw-TCP backend for the PROXY-protocol e2e test (issue #677).
# Reads the first bytes of each connection and reflects whether a HAProxy
# PROXY protocol header was prepended by gost:
#   - v1 ("PROXY TCP4 ...")  -> "PROXY-RECEIVED: PROXY TCP4 ..."
#   - v2 (12-byte signature)  -> "PROXY-V2-RECEIVED"
#   - neither                  -> "NO-PROXY"
import socket

s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
s.bind(("0.0.0.0", 5678))
s.listen(5)

while True:
    c, _ = s.accept()
    d = c.recv(4096)
    if d[:12] == b"\r\n\r\n\x00\r\nQUIT\n":
        c.sendall(b"PROXY-V2-RECEIVED\n")
    elif d[:6] == b"PROXY ":
        c.sendall(b"PROXY-RECEIVED: " + d.split(b"\r\n", 1)[0] + b"\n")
    else:
        c.sendall(b"NO-PROXY\n")
    c.close()
