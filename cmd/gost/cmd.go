package main

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/go-gost/gost/pkg/config"
)

var (
	ErrInvalidService = errors.New("invalid service")
	ErrInvalidNode    = errors.New("invalid node")
)

type stringList []string

func (l *stringList) String() string {
	return fmt.Sprintf("%s", *l)
}
func (l *stringList) Set(value string) error {
	*l = append(*l, value)
	return nil
}

func buildConfigFromCmd(services, nodes stringList) (*config.Config, error) {
	cfg := &config.Config{}

	var chain *config.ChainConfig
	if len(nodes) > 0 {
		chain = &config.ChainConfig{
			Name: "chain-0",
		}
		cfg.Chains = append(cfg.Chains, chain)
	}

	for i, node := range nodes {
		url, err := checkCmd(node)
		if err != nil {
			return nil, err
		}
		chain.Hops = append(chain.Hops, &config.HopConfig{
			Name: fmt.Sprintf("hop-%d", i),
			Nodes: []*config.NodeConfig{
				{
					Name: "node-0",
					URL:  url,
				},
			},
		})
	}

	for i, svc := range services {
		url, err := checkCmd(svc)
		if err != nil {
			return nil, err
		}
		service := &config.ServiceConfig{
			Name: fmt.Sprintf("service-%d", i),
			URL:  url,
		}
		if chain != nil {
			service.Chain = chain.Name
		}
		cfg.Services = append(cfg.Services, service)
	}

	return cfg, nil
}

func checkCmd(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ErrInvalidService
	}

	if !strings.Contains(s, "://") {
		s = "auto://" + s
	}

	u, err := url.Parse(s)
	if err != nil {
		return "", err
	}

	return u.String(), nil
}
