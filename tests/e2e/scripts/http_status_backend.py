"""HTTP server that always returns a fixed status code."""
import sys
from http.server import BaseHTTPRequestHandler, HTTPServer


STATUS = int(sys.argv[1]) if len(sys.argv) > 1 else 429
PORT = int(sys.argv[2]) if len(sys.argv) > 2 else 5680
BODY = f"status-{STATUS}".encode()


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(STATUS)
        self.send_header("Content-Length", str(len(BODY)))
        self.end_headers()
        self.wfile.write(BODY)

    def do_POST(self):
        self.send_response(STATUS)
        self.send_header("Content-Length", str(len(BODY)))
        self.end_headers()
        self.wfile.write(BODY)

    # HTTP proxy CONNECT method
    def do_CONNECT(self):
        self.send_response(STATUS)
        self.send_header("Content-Length", str(len(BODY)))
        self.end_headers()
        self.wfile.write(BODY)

    def log_message(self, format, *args):
        return


HTTPServer(("0.0.0.0", PORT), Handler).serve_forever()
