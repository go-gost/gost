#!/usr/bin/env python3
import os
import subprocess
import ssl
from http.server import BaseHTTPRequestHandler, HTTPServer

PORT = int(os.environ.get("PORT", "8443"))

cert_dir = "/tmp/certs"
os.makedirs(cert_dir, exist_ok=True)
cert_file = os.path.join(cert_dir, "cert.pem")
key_file = os.path.join(cert_dir, "key.pem")

if not os.path.exists(cert_file) or not os.path.exists(key_file):
    subprocess.run(
        [
            "openssl", "req", "-x509", "-newkey", "rsa:2048",
            "-keyout", key_file, "-out", cert_file,
            "-days", "365", "-nodes",
            "-subj", "/CN=localhost",
        ],
        check=True,
    )


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        body = b"hello-gost"
        self.send_response(200)
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, format, *args):
        return


server = HTTPServer(("0.0.0.0", PORT), Handler)
ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
ctx.load_cert_chain(cert_file, key_file)
server.socket = ctx.wrap_socket(server.socket, server_side=True)
server.serve_forever()
