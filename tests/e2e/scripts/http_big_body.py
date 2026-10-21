"""HTTP server that streams a large body.

The body is much larger than the QUIC flow control window, so a client
that stops reading fills the window of the connection that carries it.
That is what a browser does when the user cancels a download, and it is
what the flow control credit leak needs to reproduce.

Usage: http_big_body.py [port] [total_bytes] [chunk_bytes] [delay_seconds]
"""
import sys
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

PORT = int(sys.argv[1]) if len(sys.argv) > 1 else 5679
TOTAL = int(sys.argv[2]) if len(sys.argv) > 2 else 4 * 1024 * 1024
CHUNK = int(sys.argv[3]) if len(sys.argv) > 3 else 64 * 1024
DELAY = float(sys.argv[4]) if len(sys.argv) > 4 else 0.01


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.send_header("Content-Length", str(TOTAL))
        self.send_header("Content-Type", "application/octet-stream")
        self.end_headers()
        chunk = b"x" * CHUNK
        sent = 0
        try:
            while sent < TOTAL:
                n = min(CHUNK, TOTAL - sent)
                self.wfile.write(chunk[:n])
                self.wfile.flush()
                sent += n
                if DELAY > 0:
                    time.sleep(DELAY)
        except (BrokenPipeError, ConnectionResetError, OSError):
            # The client aborted the download.
            pass

    def log_message(self, format, *args):
        return


ThreadingHTTPServer(("0.0.0.0", PORT), Handler).serve_forever()
