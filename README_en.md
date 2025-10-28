# GO Simple Tunnel

### A simple security tunnel written in golang

[![en](https://img.shields.io/badge/English%20README-green)](README_en.md) [![zh](https://img.shields.io/badge/Chinese%20README-gray)](README.md)

## Features

- [x] [Listening on multiple ports](https://gost.run/en/getting-started/quick-start/)
- [x] [Multi-level forwarding chain](https://gost.run/en/concepts/chain/)
- [x] Rich protocol
- [x] [TCP/UDP port forwarding](https://gost.run/en/tutorials/port-forwarding/)
- [x] [Reverse Proxy](https://gost.run/en/tutorials/reverse-proxy/) and [Tunnel](https://gost.run/en/tutorials/reverse-proxy-tunnel/)
- [x] [TCP/UDP transparent proxy](https://gost.run/en/tutorials/redirect/)
- [x] DNS [resolver](https://gost.run/en/concepts/resolver/) and [proxy](https://gost.run/en/tutorials/dns/)
- [x] [TUN/TAP device](https://gost.run/en/tutorials/tuntap/) and [TUN2SOCKS](https://gost.run/en/tutorials/tungo/)
- [x] [Load balancing](https://gost.run/en/concepts/selector/)
- [x] [Routing control](https://gost.run/en/concepts/bypass/)
- [x] [Admission control](https://gost.run/en/concepts/limiter/)
- [x] [Bandwidth/Rate Limiter](https://gost.run/en/concepts/limiter/)
- [x] [Plugin System](https://gost.run/en/concepts/plugin/)
- [x] [Prometheus metrics](https://gost.run/en/tutorials/metrics/)
- [x] [Dynamic configuration](https://gost.run/en/tutorials/api/config/)
- [x] [Web API](https://gost.run/en/tutorials/api/overview/)
- [x] [GUI](https://github.com/go-gost/gostctl)/[WebUI](https://github.com/go-gost/gost-ui)

## Overview

![Overview](https://gost.run/images/overview.png)

There are three main ways to use GOST as a tunnel.

### Proxy

As a proxy service to access the network, multiple protocols can be used in combination to form a forwarding chain for traffic forwarding.

![Proxy](https://gost.run/images/proxy.png)

### Port Forwarding

Mapping the port of one service to the port of another service, you can also use a combination of multiple protocols to form a forwarding chain for traffic forwarding.

![Forward](https://gost.run/images/forward.png)

### Reverse Proxy

Use tunnel and intranet penetration to expose local services behind NAT or firewall to public network for access.

![Reverse Proxy](https://gost.run/images/reverse-proxy.png)

## Installation

### Binary files

[https://github.com/go-gost/gost/releases](https://github.com/go-gost/gost/releases)

### install script

```bash
# install latest from [https://github.com/go-gost/gost/releases](https://github.com/go-gost/gost/releases)
bash <(curl -fsSL https://github.com/go-gost/gost/raw/master/install.sh) --install
```
```bash
# select version for install 
bash <(curl -fsSL https://github.com/go-gost/gost/raw/master/install.sh)
```

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

## Tools

### GUI

[go-gost/gostctl](https://github.com/go-gost/gostctl)

### WebUI

[go-gost/gost-ui](https://github.com/go-gost/gost-ui)

### Shadowsocks Android

[hamid-nazari/ShadowsocksGostPlugin](https://github.com/hamid-nazari/ShadowsocksGostPlugin)

## Support

Wiki: [https://gost.run](https://gost.run/en/)

YouTube: [https://www.youtube.com/@gost-tunnel](https://www.youtube.com/@gost-tunnel)

Telegram: [https://t.me/gogost](https://t.me/gogost)

Google group: [https://groups.google.com/d/forum/go-gost](https://groups.google.com/d/forum/go-gost)

Legacy version: [v2.gost.run](https://v2.gost.run/en/)
