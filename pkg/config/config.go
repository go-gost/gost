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
	CertFile   string `yaml:"certFile,omitempty"`
	KeyFile    string `yaml:"keyFile,omitempty"`
	CAFile     string `yaml:"caFile,omitempty"`
	Secure     bool   `yaml:",omitempty"`
	ServerName string `yaml:"serverName,omitempty"`
}

type AuthConfig struct {
	Username string
	Password string
}

type SelectorConfig struct {
	Strategy    string
	MaxFails    int           `yaml:"maxFails"`
	FailTimeout time.Duration `yaml:"failTimeout"`
}

type BypassConfig struct {
	Name     string
	Reverse  bool `yaml:",omitempty"`
	Matchers []string
}

type NameserverConfig struct {
	Addr     string
	Chain    string        `yaml:",omitempty"`
	Prefer   string        `yaml:",omitempty"`
	ClientIP string        `yaml:"clientIP,omitempty"`
	Hostname string        `yaml:",omitempty"`
	TTL      time.Duration `yaml:",omitempty"`
	Timeout  time.Duration `yaml:",omitempty"`
}

type ResolverConfig struct {
	Name        string
	Nameservers []NameserverConfig
}

type HostMappingConfig struct {
	IP       string
	Hostname string
	Aliases  []string `yaml:",omitempty"`
}

type HostsConfig struct {
	Name     string
	Mappings []HostMappingConfig
}

type ListenerConfig struct {
	Type     string
	Chain    string                 `yaml:",omitempty"`
	Auths    []*AuthConfig          `yaml:",omitempty"`
	TLS      *TLSConfig             `yaml:",omitempty"`
	Metadata map[string]interface{} `yaml:",omitempty"`
}

type HandlerConfig struct {
	Type     string
	Retries  int                    `yaml:",omitempty"`
	Chain    string                 `yaml:",omitempty"`
	Auths    []*AuthConfig          `yaml:",omitempty"`
	TLS      *TLSConfig             `yaml:",omitempty"`
	Metadata map[string]interface{} `yaml:",omitempty"`
}

type ForwarderConfig struct {
	Targets  []string
	Selector *SelectorConfig `yaml:",omitempty"`
}

type DialerConfig struct {
	Type     string
	Auth     *AuthConfig            `yaml:",omitempty"`
	TLS      *TLSConfig             `yaml:",omitempty"`
	Metadata map[string]interface{} `yaml:",omitempty"`
}

type ConnectorConfig struct {
	Type     string
	Auth     *AuthConfig            `yaml:",omitempty"`
	TLS      *TLSConfig             `yaml:",omitempty"`
	Metadata map[string]interface{} `yaml:",omitempty"`
}

type ServiceConfig struct {
	Name      string
	Addr      string           `yaml:",omitempty"`
	Bypass    string           `yaml:",omitempty"`
	Resolver  string           `yaml:",omitempty"`
	Hosts     string           `yaml:",omitempty"`
	Handler   *HandlerConfig   `yaml:",omitempty"`
	Listener  *ListenerConfig  `yaml:",omitempty"`
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
	Bypass   string          `yaml:",omitempty"`
	Resolver string          `yaml:",omitempty"`
	Hosts    string          `yaml:",omitempty"`
	Nodes    []*NodeConfig
}

type NodeConfig struct {
	Name      string
	Addr      string           `yaml:",omitempty"`
	Bypass    string           `yaml:",omitempty"`
	Resolver  string           `yaml:",omitempty"`
	Hosts     string           `yaml:",omitempty"`
	Connector *ConnectorConfig `yaml:",omitempty"`
	Dialer    *DialerConfig    `yaml:",omitempty"`
}

type Config struct {
	Services  []*ServiceConfig
	Chains    []*ChainConfig    `yaml:",omitempty"`
	Bypasses  []*BypassConfig   `yaml:",omitempty"`
	Resolvers []*ResolverConfig `yaml:",omitempty"`
	Hosts     []*HostsConfig    `yaml:",omitempty"`
	TLS       *TLSConfig        `yaml:",omitempty"`
	Log       *LogConfig        `yaml:",omitempty"`
	Profiling *ProfilingConfig  `yaml:",omitempty"`
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
