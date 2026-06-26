# 执行计划：HTTP Handler E2E 测试 + utils.go 重构

## 背景

从 PLAN.md 的 Step 0 和 Step 1A 开始实施。HTTP proxy 是 gost 最基础的协议，目前 e2e 测试仅覆盖 shadowsocks 和 parallel selector。本计划先重构 utils.go 提取公共 helper，然后用 helper 写 HTTP handler 测试套件，最后同步重构 shadowsocks_test.go。

## 变更文件清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `tests/e2e/utils.go` | 修改 | 新增 `RunTCPCase` / `RunUDPCase` |
| `tests/e2e/http_test.go` | 新建 | HTTP proxy 测试套件 |
| `tests/e2e/testdata/http/server.yaml` | 新建 | 无 auth 的 HTTP proxy 服务器 |
| `tests/e2e/testdata/http/server_auth.yaml` | 新建 | 带 auth 的 HTTP proxy 服务器 |
| `tests/e2e/shadowsocks_test.go` | 修改 | 重构为使用 `RunTCPCase` / `RunUDPCase` |

---

## Step 1: 修改 `utils.go` — 提取 RunTCPCase / RunUDPCase

从 `shadowsocks_test.go` 的 `runTCPCase` 和 `runUDPCase` 提取为包级函数。关键差异：参数需要传入 echo container IP 和 echo container 本身（用于 dump logs）。

### RunTCPCase 签名与逻辑

```go
// RunTCPCase runs a full TCP proxy test case:
// start server → render client config → start client → curl assertion → cleanup.
func RunTCPCase(t *testing.T, ctx context.Context, networkName, echoIP string,
    name, serverConfig, clientConfig string)
```

逻辑（从 shadowsocks_test.go:132-160 提取，一字不差）：
1. 生成 serverAlias = `name + "-server"`
2. `RunGostContainerWithOptions(ctx, networkName, serverConfig, [serverAlias], ["8388/tcp"])`
3. `defer serverC.Terminate(ctx)`
4. `RenderConfig(clientConfig, {ServerAddr: serverAlias + ":8388"})`
5. `defer os.Remove(rendered)`
6. `RunGostContainerWithPorts(ctx, networkName, rendered, "8080/tcp")`
7. `defer clientC.Terminate(ctx)`
8. `clientC.Exec(ctx, ["curl", "-v", "-s", "-x", "http://127.0.0.1:8080", "http://<echoIP>:5678"])`
9. 断言 exitCode==0 且 body 包含 `"hello-gost"`，失败时 DumpLogs

**问题：端口硬编码**。shadowsocks 用 8388 作为服务端口，但其他协议可能用不同端口。需要参数化。

### 修正设计 — 参数化端口

```go
type TCPOptions struct {
    ServerPort    string // 容器内服务端口，默认 "8388/tcp"
    ClientPort    string // 客户端代理端口，默认 "8080/tcp"
}

func RunTCPCase(t *testing.T, ctx context.Context, networkName, echoIP string,
    name, serverConfig, clientConfig string, opts ...TCPOptions)
```

当 `opts` 为空时使用默认值 `{ServerPort: "8388/tcp", ClientPort: "8080/tcp"}`。这样 shadowsocks 测试无需任何改动，而 HTTP 测试可传 `TCPOptions{ServerPort: "8080/tcp", ClientPort: "8080/tcp"}`。

等一下 — 仔细看 shadowsocks 的 server 容器暴露的是 `8388/tcp`，HTTP proxy 场景下只有一个 gost 容器（不做 server→client 链路），直接暴露 proxy 端口。

**重新思考：HTTP proxy 不需要 server+client 两容器模式。**

HTTP proxy 测试是最简单的模式 — 只需一个 gost 容器运行 HTTP proxy，然后容器内 curl 通过它访问 echo server。这和 parallel_selector_test.go 的模式一致。但为了统一框架和后续协议（SOCKS5、relay 等都需要 server+client 模式），我们应该：

- HTTP 的"无 auth"用例：**单容器模式**（同 parallel_selector）
- HTTP 的"带 auth"用例：**也可以单容器**，只需在 curl 加 `--proxy-user`

所以 HTTP proxy 不需要 server/client 分离，但 RunTCPCase helper 仍然服务于 SOCKS5/relay 等需要链路的协议。

### 最终 RunTCPCase 设计

```go
func RunTCPCase(t *testing.T, ctx context.Context, networkName, echoIP, name string,
    serverConfig, clientConfig string, serverPort, clientPort string)
```

- `serverPort`: 服务端容器暴露端口，如 `"8388/tcp"`
- `clientPort`: 客户端容器暴露端口，如 `"8080/tcp"`
- 渲染模板时用 `ServerAddr: serverAlias + ":" + strings.TrimSuffix(serverPort, "/tcp")`

### RunUDPCase 设计

```go
func RunUDPCase(t *testing.T, ctx context.Context, networkName, name string,
    serverConfig, clientConfig string, serverPort, clientPort string)
```

逻辑从 shadowsocks_test.go:76-126 提取。

---

## Step 2: 新建 `testdata/http/` 配置文件

### `testdata/http/server.yaml` — 无 auth 的 HTTP proxy

```yaml
services:
- name: http-proxy
  addr: :8080
  handler:
    type: http
  listener:
    type: tcp
```

单容器即可测试：gost 启动后在 :8080 提供 HTTP proxy 服务。

### `testdata/http/server_auth.yaml` — 带 auth 的 HTTP proxy

```yaml
services:
- name: http-proxy-auth
  addr: :8080
  handler:
    type: http
    auther: auther-0
  listener:
    type: tcp

authers:
- name: auther-0
  auths:
  - username: user
    password: pass
```

---

## Step 3: 新建 `http_test.go`

### 套件结构

```go
type HTTPSuite struct {
    suite.Suite
    ctx    context.Context
    echoC  testcontainers.Container
    echoIP string
}

func (s *HTTPSuite) SetupSuite()    // 启动 TCP echo
func (s *HTTPSuite) TearDownSuite() // 关闭 echo

func (s *HTTPSuite) TestHTTPProxy()       // 无 auth
func (s *HTTPSuite) TestHTTPProxyAuth()   // 带 auth

func TestHTTPSuite(t *testing.T) {
    suite.Run(t, new(HTTPSuite))
}
```

### TestHTTPProxy — 单容器模式（同 parallel_selector）

```go
func (s *HTTPSuite) TestHTTPProxy() {
    gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName,
        "testdata/http/server.yaml", "8080/tcp")
    s.Require().NoError(err)
    defer gostC.Terminate(s.ctx)

    cmd := []string{"curl", "-v", "-s", "-x", "http://127.0.0.1:8080",
        fmt.Sprintf("http://%s:5678", s.echoIP)}
    code, out, err := gostC.Exec(s.ctx, cmd)
    s.Require().NoError(err)

    body, err := io.ReadAll(out)
    s.Require().NoError(err)
    if code != 0 || !strings.Contains(string(body), "hello-gost") {
        DumpLogs(s.T(), s.ctx, "http-proxy logs", gostC)
    }
    s.Require().Equal(0, code)
    s.Require().Contains(string(body), "hello-gost")
}
```

### TestHTTPProxyAuth — 带 auth 验证

```go
func (s *HTTPSuite) TestHTTPProxyAuth() {
    gostC, err := RunGostContainerWithPorts(s.ctx, SharedNetworkName,
        "testdata/http/server_auth.yaml", "8080/tcp")
    s.Require().NoError(err)
    defer gostC.Terminate(s.ctx)

    // 测试1：无 auth 应失败（407）
    s.T().Run("no-auth-should-fail", func(t *testing.T) {
        cmd := []string{"curl", "-s", "-o", "/dev/null", "-w", "%{http_code}",
            "-x", "http://127.0.0.1:8080", fmt.Sprintf("http://%s:5678", s.echoIP)}
        code, out, _ := gostC.Exec(s.ctx, cmd)
        body, _ := io.ReadAll(out)
        // curl exit code 非 0 或 HTTP status 是 407
        // 注意：gost HTTP proxy 在无 auth 时返回 407，curl 不会自动重试
        // 所以这里验证返回码是 407 或 curl 返回非零
        httpStatus := strings.TrimSpace(string(body))
        s.Assert().True(code != 0 || httpStatus == "407",
            "expected auth failure, got status: %s, exit code: %d", httpStatus, code)
    })

    // 测试2：正确 auth 应成功
    s.T().Run("with-auth-should-succeed", func(t *testing.T) {
        cmd := []string{"curl", "-v", "-s", "-x",
            "http://user:pass@127.0.0.1:8080",
            fmt.Sprintf("http://%s:5678", s.echoIP)}
        code, out, err := gostC.Exec(s.ctx, cmd)
        s.Require().NoError(err)

        body, err := io.ReadAll(out)
        s.Require().NoError(err)
        if code != 0 || !strings.Contains(string(body), "hello-gost") {
            DumpLogs(s.T(), s.ctx, "http-proxy-auth logs", gostC)
        }
        s.Require().Equal(0, code)
        s.Require().Contains(string(body), "hello-gost")
    })
}
```

### 注意事项
- HTTP proxy 是**单容器模式**，不需要 server/client 分离（不像 shadowsocks 需要解密/加密链路）
- auth 测试分两步：先验证无 auth 被拒（407），再验证正确 auth 成功
- curl 使用 `http://user:pass@host:port` 格式传递 proxy auth

---

## Step 4: 重构 `shadowsocks_test.go`

将 `runTCPCase` 和 `runUDPCase` 方法替换为调用 `utils.go` 中的 `RunTCPCase` / `RunUDPCase`：

**Before**:
```go
func (s *ShadowsocksSuite) TestShadowsocksTCP() {
    s.runTCPCase("aes256gcm", "testdata/shadowsocks/tcp_server_aes256gcm.yaml", "testdata/shadowsocks/tcp_client_aes256gcm.yaml")
    s.runTCPCase("chacha20", "testdata/shadowsocks/tcp_server_chacha20.yaml", "testdata/shadowsocks/tcp_client_chacha20.yaml")
}
```

**After**:
```go
func (s *ShadowsocksSuite) TestShadowsocksTCP() {
    RunTCPCase(s.T(), s.ctx, SharedNetworkName, s.echoIP, "aes256gcm",
        "testdata/shadowsocks/tcp_server_aes256gcm.yaml",
        "testdata/shadowsocks/tcp_client_aes256gcm.yaml",
        "8388/tcp", "8080/tcp")
    RunTCPCase(s.T(), s.ctx, SharedNetworkName, s.echoIP, "chacha20",
        "testdata/shadowsocks/tcp_server_chacha20.yaml",
        "testdata/shadowsocks/tcp_client_chacha20.yaml",
        "8388/tcp", "8080/tcp")
}
```

删除 `runTCPCase` 和 `runUDPCase` 方法。UDP 测试同理。

---

## 实施顺序

1. **`utils.go`** — 添加 `RunTCPCase` + `RunUDPCase`（不删除任何现有函数）
2. **`testdata/http/`** — 创建 `server.yaml` + `server_auth.yaml`
3. **`http_test.go`** — 创建 HTTP 套件
4. **编译验证** — `cd gost && go build ./... && go vet ./tests/e2e/...`
5. **`shadowsocks_test.go`** — 重构为使用新 helper
6. **最终编译验证** — `go build ./... && go vet ./tests/e2e/...`
7. **运行测试** — `go test ./tests/e2e/ -v -run TestHTTPSuite -timeout 5m`
8. **回归测试** — `go test ./tests/e2e/ -v -run TestShadowsocksSuite -timeout 5m`

---

## 验证

```bash
cd /config/workspace/go-gost/gost
go build ./...
go vet ./tests/e2e/...
go test ./tests/e2e/ -v -run TestHTTPSuite -timeout 5m
go test ./tests/e2e/ -v -run TestShadowsocksSuite -timeout 5m
```
