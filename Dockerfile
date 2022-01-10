FROM golang:1-alpine as builder

RUN apk add --no-cache musl-dev gcc

WORKDIR /mod

ADD go.mod go.sum ./

RUN go mod download

ADD . /src

WORKDIR /src

RUN cd cmd/gost && go env && go build 

FROM alpine:latest

WORKDIR /bin/

COPY --from=builder /src/cmd/gost/gost .

ENTRYPOINT ["/bin/gost"]
