package config

import (
	"io"
	"time"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
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
	Output string `yaml:",omitempty"`
	Level  string `yaml:",omitempty"`
	Format string `yaml:",omitempty"`
}

type ProfilingConfig struct {
	Addr    string
	Enabled bool
}

type TLSConfig struct {
	Cert string
	Key  string
	CA   string
}

type SelectorConfig struct {
	Strategy    string
	MaxFails    int
	FailTimeout time.Duration
}

type BypassConfig struct {
	Name     string
	Reverse  bool `yaml:",omitempty"`
	Matchers []string
}

type NameserverConfig struct {
	Addr     string
	Chain    string
	Prefer   string
	ClientIP string
	Hostname string
	TTL      time.Duration
	Timeout  time.Duration
}

type ResolverConfig struct {
	Name        string
	Nameservers []NameserverConfig
}

type HostConfig struct {
	IP       string
	Hostname string
	Aliases  []string
}

type HostsConfig struct {
	Name    string
	Entries []HostConfig
}

type ListenerConfig struct {
	Type     string
	Metadata map[string]interface{} `yaml:",omitempty"`
}

type HandlerConfig struct {
	Type     string
	Metadata map[string]interface{} `yaml:",omitempty"`
}

type ForwarderConfig struct {
	Targets  []string
	Selector *SelectorConfig `yaml:",omitempty"`
}

type DialerConfig struct {
	Type     string
	Metadata map[string]interface{} `yaml:",omitempty"`
}

type ConnectorConfig struct {
	Type     string
	Metadata map[string]interface{} `yaml:",omitempty"`
}

type ServiceConfig struct {
	Name      string
	URL       string           `yaml:",omitempty"`
	Addr      string           `yaml:",omitempty"`
	Chain     string           `yaml:",omitempty"`
	Bypass    string           `yaml:",omitempty"`
	Resolver  string           `yaml:",omitempty"`
	Hosts     string           `yaml:",omitempty"`
	Listener  *ListenerConfig  `yaml:",omitempty"`
	Handler   *HandlerConfig   `yaml:",omitempty"`
	Forwarder *ForwarderConfig `yaml:",omitempty"`
}

type ChainConfig struct {
	Name     string
	Selector *SelectorConfig `yaml:",omitempty"`
	Hops     []*HopConfig
}

type HopConfig struct {
	Name     string
	Selector *SelectorConfig `yaml:",omitempty"`
	Nodes    []*NodeConfig
}

type NodeConfig struct {
	Name      string
	URL       string           `yaml:",omitempty"`
	Addr      string           `yaml:",omitempty"`
	Dialer    *DialerConfig    `yaml:",omitempty"`
	Connector *ConnectorConfig `yaml:",omitempty"`
	Bypass    string           `yaml:",omitempty"`
}

type Config struct {
	Log       *LogConfig        `yaml:",omitempty"`
	Profiling *ProfilingConfig  `yaml:",omitempty"`
	TLS       *TLSConfig        `yaml:",omitempty"`
	Bypasses  []*BypassConfig   `yaml:",omitempty"`
	Resolvers []*ResolverConfig `yaml:",omitempty"`
	Hosts     []*HostsConfig    `yaml:",omitempty"`
	Chains    []*ChainConfig    `yaml:",omitempty"`
	Services  []*ServiceConfig
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

func (c *Config) Write(w io.Writer) error {
	enc := yaml.NewEncoder(w)
	defer enc.Close()

	return enc.Encode(c)
}
