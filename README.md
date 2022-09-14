# GO Simple Tunnel

### GO语言实现的安全隧道

[English README](README_en.md)

## 功能特性

- [x] [多端口监听](https://gost.run/getting-started/quick-start/)
- [x] [多级转发链](https://gost.run/concepts/chain/)
- [x] 多协议支持
- [x] [TCP/UDP端口转发](https://gost.run/tutorials/port-forwarding/)
- [x] [TCP/UDP透明代理](https://gost.run/tutorials/redirect/)
- [x] DNS[解析](https://gost.run/concepts/resolver/)和[代理](https://gost.run/tutorials/dns/)
- [x] [TUN/TAP设备](https://gost.run/tutorials/tuntap/)
- [x] [负载均衡](https://gost.run/concepts/selector/)
- [x] [路由控制](https://gost.run/concepts/bypass/)
- [x] [限速限流](https://gost.run/concepts/limiter/)
- [x] [准入控制](https://gost.run/concepts/admission/)
- [x] [动态配置](https://gost.run/tutorials/api/config/)
- [x] [Prometheus监控指标](https://gost.run/tutorials/metrics/)
- [x] [Web API](https://gost.run/tutorials/api/overview/)
- [ ] Web UI

Wiki站点：[https://gost.run](https://gost.run)

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
