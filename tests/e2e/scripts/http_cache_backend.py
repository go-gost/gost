from http.server import BaseHTTPRequestHandler, HTTPServer

_counter = 0


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        global _counter
        _counter += 1
        body = b"cache-test-%d" % _counter
        self.send_response(200)
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, format, *args):
        return


HTTPServer(("0.0.0.0", 5677), Handler).serve_forever()
