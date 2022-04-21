# GO Simple Tunnel

### GO语言实现的安全隧道

[English README](README_en.md)

## 功能特性

- [x] 多端口监听
- [x] 多级转发链
- [x] 多协议支持
- [x] TCP/UDP端口转发
- [x] TCP/UDP透明代理
- [x] DNS解析和代理
- [x] TUN/TAP设备
- [x] 负载均衡
- [x] 路由控制
- [x] 准入控制
- [x] 动态配置
- [x] Prometheus监控指标
- [x] Web API
- [ ] Web UI

Wiki站点：[https://latest.gost.run](https://latest.gost.run)

Telegram讨论群：[https://t.me/gogost](https://t.me/gogost)

Google讨论组：[https://groups.google.com/d/forum/go-gost](https://groups.google.com/d/forum/go-gost)

旧版入口：[v2.gost.run](https://v2.gost.run)

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
