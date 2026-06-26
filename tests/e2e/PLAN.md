# Plan: Expand E2E Test Suite

## Context

The e2e test framework (`tests/e2e/`) currently covers only **2 protocols** (Shadowsocks, parallel selector) out of **~30+ registered protocols** in gost. This plan systematically adds test coverage for all major protocols, following the existing Docker-based testcontainers pattern exactly.

**Key insight**: gost auto-generates self-signed TLS certs when none are provided (`x/config/parsing/tls.go` — `BuildDefaultTLSConfig`), and clients with default config skip cert verification. This means TLS-based protocols (ws→wss, h2c→h2, quic, etc.) can be tested **without mounting cert files**.

---

## Step 0: Refactor `utils.go` — Add Reusable Test Helpers

**File**: `tests/e2e/utils.go`

Extract the repeated patterns from `shadowsocks_test.go` into reusable helpers:

1. **`RunTCPCase(t, ctx, networkName, echoIP, name, serverConfig, clientConfig string)`** — Full lifecycle: start server → render client config → start client → curl assertion → cleanup. Extracts the logic from `ShadowsocksSuite.runTCPCase`.

2. **`RunUDPCase(t, ctx, networkName, name, serverConfig, clientConfig string)`** — Same for UDP: start server → render client → start client → dial UDP → retry loop → assertion → cleanup. Extracts from `runUDPCase`.

These eliminate ~40 lines of boilerplate per suite. Each suite's test methods become one-liners:
```go
func (s *SOCKS5Suite) TestSOCKS5TCP() {
    RunTCPCase(s.T(), s.ctx, SharedNetworkName, s.echoIP, "socks5-tcp",
        "testdata/socks5/tcp_server.yaml", "testdata/socks5/tcp_client.yaml")
}
```

---

## Step 1: Tier 1 — Core Protocols (Plain TCP, No Special Infrastructure)

Each suite follows the exact pattern: suite struct → `SetupSuite` (echo containers) → `TearDownSuite` → test methods calling `RunTCPCase`/`RunUDPCase` → `suite.Run()` entry point.

### 1A. HTTP Proxy Suite
- **Files**: `http_test.go`, `testdata/http/` (2 configs)
- **Configs**: `server.yaml` (handler `http`, listener `tcp`), `server_auth.yaml` (+ basic auth)
- **Tests**: `TestHTTPProxy`, `TestHTTPProxyAuth` (curl with `--proxy-user`)
- **Note**: Simplest test — single gost container acts as HTTP proxy

### 1B. SOCKS5 Suite
- **Files**: `socks5_test.go`, `testdata/socks5/` (4 configs)
- **Configs**: `tcp_server.yaml` (handler `socks5`), `tcp_client.yaml` (connector `socks5`, dialer `tcp`), + auth variants
- **Tests**: `TestSOCKS5TCP`, `TestSOCKS5TCPAuth`

### 1C. SOCKS4 Suite
- **Files**: `socks4_test.go`, `testdata/socks4/` (4 configs)
- **Configs**: `tcp_server.yaml` (handler `socks4`), `tcp_client.yaml` (connector `socks4`), + socks4a variants
- **Tests**: `TestSOCKS4TCP`, `TestSOCKS4aTCP`

### 1D. HTTP2 Cleartext Suite
- **Files**: `http2_test.go`, `testdata/http2/` (2 configs)
- **Configs**: `tcp_server.yaml` (handler `http2`, listener `http2`), `tcp_client.yaml` (connector `http2`, dialer `http2`)
- **Tests**: `TestHTTP2TCP`

### 1E. Relay Suite
- **Files**: `relay_test.go`, `testdata/relay/` (4 configs)
- **Configs**: `tcp_server.yaml` (handler `relay`, listener `tcp`), `tcp_client.yaml` (connector `relay`, dialer `tcp`), + auth variants
- **Tests**: `TestRelayTCP`, `TestRelayTCPAuth`

---

## Step 2: Tier 2 — TLS/Transport Protocols

Same pattern but using TLS listeners/dialers. No explicit cert files needed — gost auto-generates them.

### 2A. TLS Suite
- **Files**: `tls_test.go`, `testdata/tls/` (4 configs)
- **Tests**: `TestTLSHTTPProxy` (handler `http` + listener `tls`), `TestTLSSOCKS5` (handler `socks5` + listener `tls`)

### 2B. WebSocket Suite
- **Files**: `ws_test.go`, `testdata/ws/` (4 configs)
- **Tests**: `TestWSSOCKS5` (dialer `ws`), `TestWSSSOCKS5` (dialer `wss`)

### 2C. gRPC Suite
- **Files**: `grpc_test.go`, `testdata/grpc/` (2 configs)
- **Tests**: `TestGRPCSOCKS5` (listener `grpc`, metadata `insecure: true`)

### 2D. HTTP/2 TLS (h2) Suite
- **Files**: `h2_test.go`, `testdata/h2/` (2 configs)
- **Tests**: `TestH2HTTP2Proxy` (listener `h2`, dialer `h2`)

### 2E. QUIC Suite
- **Files**: `quic_test.go`, `testdata/quic/` (2 configs)
- **Tests**: `TestQUICHTTP` (listener `quic`, handler `http` — uses UDP ports)

### 2F. KCP Suite
- **Files**: `kcp_test.go`, `testdata/kcp/` (2 configs)
- **Tests**: `TestKCPSOCKS5` (listener `kcp`, handler `socks5` — uses UDP)

### 2G. Multiplex Variants (mws/mwss/mtls/mtcp)
- **Files**: `mux_test.go`, `testdata/mux/` (6 configs)
- **Tests**: `TestMWSSOCKS5`, `TestMWSSSOCKS5`, `TestMtlSSOCKS5`, `TestMtcpSOCKS5`

### 2H. Obfuscation (ohttp/otls)
- **Files**: `obfs_test.go`, `testdata/obfs/` (4 configs)
- **Tests**: `TestOHTTP`, `TestOTLS`

### 2I. PHT (HTTP Pipe)
- **Files**: `pht_test.go`, `testdata/pht/` (4 configs)
- **Tests**: `TestPHT`, `TestPHTS`

---

## Step 3: Tier 3 — Special Infrastructure (Deferred)

These require Docker capabilities (`NET_ADMIN`, `NET_RAW`), SSH key generation, or iptables setup. Implemented after Tiers 1-2 are stable.

| Protocol | Challenge | Approach |
|----------|-----------|----------|
| SSH/sshd | Host key generation | Generate SSH key at test time, mount into container |
| HTTP/3/WebTransport | QUIC+UDP, TLS | Uses UDP like QUIC suite; auto certs work |
| TUN/TAP | `CAP_NET_ADMIN`, `/dev/net/tun` | Privileged container + routing setup |
| Tungo/MASQUE | Same as TUN | Same approach |
| ICMP | `CAP_NET_RAW`, raw sockets | Privileged container |
| Redirect (red/redu) | iptables rules | Privileged + iptables setup in container |
| DTLS | UDP + TLS | Auto certs; UDP port mapping |

---

## Files to Create/Modify

### New files (per suite):
- `tests/e2e/<protocol>_test.go` — Suite struct + test methods
- `tests/e2e/testdata/<protocol>/` — Server + client YAML configs

### Modified files:
- `tests/e2e/utils.go` — Add `RunTCPCase`, `RunUDPCase` helpers
- `tests/e2e/shadowsocks_test.go` — Refactor to use new helpers (optional, reduces duplication)
- `tests/e2e/README.md` — Document new suites

### Reference files (read-only):
- `tests/e2e/shadowsocks_test.go` — Pattern template for all suites
- `tests/e2e/testdata/shadowsocks/tcp_server_aes256gcm.yaml` — Server config template
- `tests/e2e/testdata/shadowsocks/tcp_client_aes256gcm.yaml` — Client config template

---

## Config Pattern Reference

**Server** (all protocols):
```yaml
services:
- name: <proto>-server
  addr: :<port>
  handler:
    type: <handler-type>
  listener:
    type: <listener-type>
```

**Client** (all protocols):
```yaml
services:
- name: http-proxy
  addr: :8080
  handler:
    type: http
    chain: <proto>-chain
  listener:
    type: tcp
chains:
- name: <proto>-chain
  hops:
  - name: <proto>-hop
    nodes:
    - name: <proto>-node
      addr: {{.ServerAddr}}
      connector:
        type: <connector-type>
      dialer:
        type: <dialer-type>
```

---

## Verification

After each suite is implemented:
1. `cd gost && go build ./...` — Ensure compilation
2. `go vet ./tests/e2e/` — Static analysis
3. `go test ./tests/e2e/ -v -run TestXxxSuite -timeout 5m` — Run individual suite
4. `go test ./tests/e2e/ -v -timeout 10m` — Run all suites together
