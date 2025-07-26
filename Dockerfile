FROM --platform=$BUILDPLATFORM tonistiigi/xx:1.6.1 AS xx

FROM --platform=$BUILDPLATFORM golang:1.24-alpine3.22 AS builder

# add upx for binary compression
RUN apk add --no-cache upx || echo "upx not found"

COPY --from=xx / /

ARG TARGETPLATFORM

RUN xx-info env

ENV CGO_ENABLED=0

ENV XX_VERIFY_STATIC=1

WORKDIR /app

COPY . .

RUN cd cmd/gost && \
    xx-go build -ldflags "-s -w" && \
    xx-verify gost && \
    { upx --best gost || true; }

FROM alpine:3.22

# add iptables for tun/tap
RUN apk add --no-cache iptables

WORKDIR /bin/

COPY --from=builder /app/cmd/gost/gost .

ENTRYPOINT ["/bin/gost"]