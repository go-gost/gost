# GO Simple Tunnel

## GO语言实现的安全隧道

## 功能特性

- [x] 多端口监听
- [x] 支持转发链，并支持多级转发
- [x] 支持多种协议(HTTP，HTTPS，HTTP2，SOCKS5，Websocket，QUIC...)
- [x] 本地/远程TCP/UDP端口转发
- [x] DNS解析和代理
- [x] TUN/TAP设备
- [x] 负载均衡
- [x] 路由控制
- [x] 动态配置
- [x] Prometheus Metrics
- [x] Web API
- [ ] Web UI

## 下载安装

### 二进制文件

[https://github.com/go-gost/gost/releases](https://github.com/go-gost/gost/releases)

### 源码编译

```
git clone https://github.com/go-gost/gost.git
cd gost/cmd/gost
go build
```

### Docker

```
docker pull gogost/gost
```

### Shadowsocks Android插件

[xausky/ShadowsocksGostPlugin](https://github.com/xausky/ShadowsocksGostPlugin)

## 问题建议

提交Issue: [https://github.com/go-gost/gost/issues](https://github.com/go-gost/gost/issues)

Telegram讨论群: [https://t.me/gogost](https://t.me/gogost)

Google讨论组: [https://groups.google.com/d/forum/go-gost](https://groups.google.com/d/forum/go-gost)