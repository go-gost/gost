import socket, threading

s = socket.socket()
s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
s.bind(("127.0.0.1", 15738))
s.listen(50)

def loop(c):
    try:
        while True:
            d = c.recv(65536)
            if not d:
                break
            c.sendall(d)
    except OSError:
        pass
    finally:
        c.close()

while True:
    c, _ = s.accept()
    threading.Thread(target=loop, args=(c,), daemon=True).start()
