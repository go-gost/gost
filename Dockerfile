FROM --platform=$BUILDPLATFORM tonistiigi/xx:1.6.1 AS xx

FROM --platform=$BUILDPLATFORM golang:1.26-alpine3.23 AS builder

# UPX compression disabled by default (see #863): upx --best adds ~3s startup
# time on low-spec systems (linux/arm). Builds can opt in by uncommenting the
# upx install line and adding upx --best to the build command below.
# RUN apk add --no-cache upx || echo "upx not found"

COPY --from=xx / /

ARG TARGETPLATFORM

RUN xx-info env

ENV CGO_ENABLED=0

ENV XX_VERIFY_STATIC=1

WORKDIR /app

COPY . .

RUN cd cmd/gost && \
    xx-go build -ldflags "-s -w" && \
    xx-verify gost

FROM alpine:3.23

# add iptables/nftables for tun/tap
RUN apk add --no-cache iptables nftables

WORKDIR /bin/

COPY --from=builder /app/cmd/gost/gost .

ENTRYPOINT ["/bin/gost"]