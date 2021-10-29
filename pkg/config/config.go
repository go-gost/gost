package config

import (
	"io"

	"github.com/spf13/viper"
)

var (
	v = viper.GetViper()
)

func init() {
	v.SetConfigName("gost")
	v.AddConfigPath("/etc/gost/")
	v.AddConfigPath("$HOME/.gost/")
	v.AddConfigPath(".")
}

type LogConfig struct {
	Output string
	Level  string
	Format string
}

type LoadbalancingConfig struct {
	Strategy string
	Filters  []string
}

type ListenerConfig struct {
	Type     string
	Metadata map[string]interface{}
}

type HandlerConfig struct {
	Type     string
	Metadata map[string]interface{}
}

type DialerConfig struct {
	Type     string
	Metadata map[string]interface{}
}

type ConnectorConfig struct {
	Type     string
	Metadata map[string]interface{}
}

type ServiceConfig struct {
	URL      string
	Addr     string
	Listener *ListenerConfig
	Handler  *HandlerConfig
	Chain    string
}

type ChainConfig struct {
	Name string
	LB   *LoadbalancingConfig
	Hops []HopConfig
}

type HopConfig struct {
	Name  string
	LB    *LoadbalancingConfig
	Nodes []NodeConfig
}

type NodeConfig struct {
	Name      string
	URL       string
	Addr      string
	Dialer    *DialerConfig
	Connector *ConnectorConfig
}

type Config struct {
	Log      *LogConfig
	Services []ServiceConfig
	Chains   []ChainConfig
}

func (c *Config) Load() error {
	if err := v.ReadInConfig(); err != nil {
		return err
	}

	return v.Unmarshal(c)
}

func (c *Config) Read(r io.Reader) error {
	if err := v.ReadConfig(r); err != nil {
		return err
	}

	return v.Unmarshal(c)
}

func (c *Config) ReadFile(file string) error {
	v.SetConfigFile(file)
	if err := v.ReadInConfig(); err != nil {
		return err
	}
	return v.Unmarshal(c)
}
