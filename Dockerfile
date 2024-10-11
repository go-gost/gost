FROM --platform=$BUILDPLATFORM golang:1.23-alpine3.20 AS builder

ARG TARGETOS
ARG TARGETARCH

# RUN apk add --no-cache musl-dev git gcc

ENV CGO_ENABLED=0

RUN go env

WORKDIR /app

# Cache the download before continuing
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY . .

WORKDIR /app/cmd/gost

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build

FROM alpine:3.20

# add iptables for tun/tap
RUN apk add --no-cache iptables

WORKDIR /bin/

COPY --from=builder /app/cmd/gost/gost .

ENTRYPOINT ["/bin/gost"]