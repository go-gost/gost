import http.server
import json
import sys


class AuthHandler(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        length = int(self.headers.get('Content-Length', 0))
        body = self.rfile.read(length)
        data = json.loads(body)
        print(f"AUTH_REQUEST: {json.dumps(data)}", flush=True)
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(
            json.dumps(
                {"ok": True, "id": data.get("username", "anonymous")}
            ).encode()
        )

    def log_message(self, format, *args):
        pass  # silence default logging, we use our own AUTH_REQUEST line


if __name__ == "__main__":
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 9000
    http.server.HTTPServer(("0.0.0.0", port), AuthHandler).serve_forever()
