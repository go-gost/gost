# GO Simple Tunnel

### A simple security tunnel written in golang

## Features

- [x] [Listening on multiple ports](https://gost.run/en/getting-started/quick-start/)
- [x] [Multi-level forwarding chain](https://gost.run/en/concepts/chain/)
- [x] Rich protocol
- [x] [TCP/UDP port forwarding](https://gost.run/en/tutorials/port-forwarding/)
- [x] [Reverse Proxy](https://gost.run/en/tutorials/reverse-proxy/) and [Tunnel](https://gost.run/en/tutorials/reverse-proxy-advanced/)
- [x] [TCP/UDP transparent proxy](https://gost.run/en/tutorials/redirect/)
- [x] DNS [resolver](https://gost.run/en/concepts/resolver/) and [proxy](https://gost.run/en/tutorials/dns/)
- [x] [TUN/TAP device](https://gost.run/en/tutorials/tuntap/)
- [x] [Load balancing](https://gost.run/en/concepts/selector/)
- [x] [Routing control](https://gost.run/en/concepts/bypass/)
- [x] [Admission control](https://gost.run/en/concepts/limiter/)
- [x] [Bandwidth/Rate Limiter](https://gost.run/en/concepts/limiter/)
- [x] [Plugin System](https://gost.run/en/concepts/plugin/)
- [x] [Prometheus metrics](https://gost.run/en/tutorials/metrics/)
- [x] [Dynamic configuration](https://gost.run/en/tutorials/api/config/)
- [x] [Web API](https://gost.run/en/tutorials/api/overview/)
- [ ] Web UI

Wiki: [https://gost.run](https://gost.run/en/)

Telegram: [https://t.me/gogost](https://t.me/gogost)

Google group: [https://groups.google.com/d/forum/go-gost](https://groups.google.com/d/forum/go-gost)

Legacy version: [v2.gost.run](https://v2.gost.run/en/)

## Installation


### Binary files

[https://github.com/go-gost/gost/releases](https://github.com/go-gost/gost/releases)

### From source

```
git clone https://github.com/go-gost/gost.git
cd gost/cmd/gost
go build
```

### Docker

```
docker run --rm gogost/gost -V
```

### Shadowsocks Android

[xausky/ShadowsocksGostPlugin](https://github.com/xausky/ShadowsocksGostPlugin)
