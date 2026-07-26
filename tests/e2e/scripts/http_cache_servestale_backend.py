from http.server import BaseHTTPRequestHandler, HTTPServer
import socket
import threading

_counter = 0
_http_server = None


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        global _counter
        _counter += 1
        body = b"cache-test-%d" % _counter
        self.send_response(200)
        self.send_header("Content-Length", str(len(body)))
        self.send_header("Connection", "close")
        self.end_headers()
        self.wfile.write(body)
        threading.Thread(target=stop_server, daemon=True).start()

    def log_message(self, format, *args):
        return


def stop_server():
    global _http_server
    if _http_server:
        _http_server.shutdown()
        _http_server.server_close()


def start_raw_acceptor():
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    sock.bind(("0.0.0.0", 5676))
    sock.listen(5)
    while True:
        conn, addr = sock.accept()
        conn.close()


_http_server = HTTPServer(("0.0.0.0", 5676), Handler)
_http_server.serve_forever()
_http_server.server_close()

start_raw_acceptor()
