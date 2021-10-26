package main

import (
	"bufio"
	"log"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/go-gost/gost/pkg/chain"
	"github.com/go-gost/gost/pkg/components/connector"
	httpc "github.com/go-gost/gost/pkg/components/connector/http"
	tcpd "github.com/go-gost/gost/pkg/components/dialer/tcp"
	"github.com/go-gost/gost/pkg/components/handler"
	httph "github.com/go-gost/gost/pkg/components/handler/http"
	"github.com/go-gost/gost/pkg/components/listener"
	tcpl "github.com/go-gost/gost/pkg/components/listener/tcp"
	"github.com/go-gost/gost/pkg/service"
	"golang.org/x/net/context"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	testChain()
	testHTTPHandler()
}

func testHTTPHandler() {
	ln := tcpl.NewListener()
	if err := ln.Init(listener.Metadata{
		"addr": ":1080",
	}); err != nil {
		log.Fatal(err)
	}

	chain := createChain()
	h := httph.NewHandler(
		handler.ChainOption(chain),
	)
	svc := (&service.Service{}).
		WithListener(ln).
		WithHandler(h)
	log.Fatal(svc.Run())
}

func createChain() *chain.Chain {
	c := httpc.NewConnector()
	c.Init(connector.Metadata{
		//"userAgent": "gost-client-3",
		"username": "admin",
		"password": "123456",
	})
	tr := (&chain.Transport{}).
		WithDialer(tcpd.NewDialer()).
		WithConnector(c)

	node := chain.NewNode("local", "localhost:8080").
		WithTransport(tr)

	ch := &chain.Chain{}
	ch.AddNodeGroup(chain.NewNodeGroup(node))

	return ch
}

func testChain() {
	chain := createChain()
	r := chain.GetRoute()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := r.Dial(ctx, "tcp", "www.google.com:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	log.Println(conn.LocalAddr(), conn.RemoteAddr())

	req, err := http.NewRequest("GET", "http://www.google.com", nil)
	if err != nil {
		log.Fatal(err)
	}
	if err := req.Write(conn); err != nil {
		log.Fatal(err)
	}
	data, _ := httputil.DumpRequest(req, true)
	log.Println(string(data))

	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	data, _ = httputil.DumpResponse(resp, true)
	log.Println(string(data))
}
