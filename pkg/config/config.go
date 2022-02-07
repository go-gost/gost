package config

import (
	"encoding/json"
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
	Output string `yaml:",omitempty" json:"output,omitempty"`
	Level  string `yaml:",omitempty" json:"level,omitempty"`
	Format string `yaml:",omitempty" json:"format,omitempty"`
}

type ProfilingConfig struct {
	Addr    string `json:"addr"`
	Enabled bool   `json:"enabled"`
}

type TLSConfig struct {
	CertFile   string `yaml:"certFile,omitempty" json:"certFile,omitempty"`
	KeyFile    string `yaml:"keyFile,omitempty" json:"keyFile,omitempty"`
	CAFile     string `yaml:"caFile,omitempty" json:"caFile,omitempty"`
	Secure     bool   `yaml:",omitempty" json:"secure,omitempty"`
	ServerName string `yaml:"serverName,omitempty" json:"serverName,omitempty"`
}

type AuthConfig struct {
	Username string `json:"username"`
	Password string `yaml:",omitempty" json:"password,omitempty"`
}

type SelectorConfig struct {
	Strategy    string        `json:"strategy"`
	MaxFails    int           `yaml:"maxFails" json:"maxFails"`
	FailTimeout time.Duration `yaml:"failTimeout" json:"failTimeout"`
}

type BypassConfig struct {
	Name     string   `json:"name"`
	Reverse  bool     `yaml:",omitempty" json:"reverse,omitempty"`
	Matchers []string `json:"matchers"`
}

type NameserverConfig struct {
	Addr     string        `json:"addr"`
	Chain    string        `yaml:",omitempty" json:"chain,omitempty"`
	Prefer   string        `yaml:",omitempty" json:"prefer,omitempty"`
	ClientIP string        `yaml:"clientIP,omitempty" json:"clientIP,omitempty"`
	Hostname string        `yaml:",omitempty" json:"hostname,omitempty"`
	TTL      time.Duration `yaml:",omitempty" json:"ttl,omitempty"`
	Timeout  time.Duration `yaml:",omitempty" json:"timeout,omitempty"`
}

type ResolverConfig struct {
	Name        string             `json:"name"`
	Nameservers []NameserverConfig `json:"nameservers"`
}

type HostMappingConfig struct {
	IP       string   `json:"ip"`
	Hostname string   `json:"hostname"`
	Aliases  []string `yaml:",omitempty" json:"aliases,omitempty"`
}

type HostsConfig struct {
	Name     string              `json:"name"`
	Mappings []HostMappingConfig `json:"mappings"`
}

type ListenerConfig struct {
	Type     string                 `json:"type"`
	Chain    string                 `yaml:",omitempty" json:"chain,omitempty"`
	Auths    []*AuthConfig          `yaml:",omitempty" json:"auths,omitempty"`
	TLS      *TLSConfig             `yaml:",omitempty" json:"tls,omitempty"`
	Metadata map[string]interface{} `yaml:",omitempty" json:"metadata,omitempty"`
}

type HandlerConfig struct {
	Type     string                 `json:"type"`
	Retries  int                    `yaml:",omitempty" json:"retries,omitempty"`
	Chain    string                 `yaml:",omitempty" json:"chain,omitempty"`
	Auths    []*AuthConfig          `yaml:",omitempty" json:"auths,omitempty"`
	TLS      *TLSConfig             `yaml:",omitempty" json:"tls,omitempty"`
	Metadata map[string]interface{} `yaml:",omitempty" json:"metadata,omitempty"`
}

type ForwarderConfig struct {
	Targets  []string        `json:"targets"`
	Selector *SelectorConfig `yaml:",omitempty" json:"selector,omitempty"`
}

type DialerConfig struct {
	Type     string                 `json:"type"`
	Auth     *AuthConfig            `yaml:",omitempty" json:"auth,omitempty"`
	TLS      *TLSConfig             `yaml:",omitempty" json:"tls,omitempty"`
	Metadata map[string]interface{} `yaml:",omitempty" json:"metadata,omitempty"`
}

type ConnectorConfig struct {
	Type     string                 `json:"type"`
	Auth     *AuthConfig            `yaml:",omitempty" json:"auth,omitempty"`
	TLS      *TLSConfig             `yaml:",omitempty" json:"tls,omitempty"`
	Metadata map[string]interface{} `yaml:",omitempty" json:"metadata,omitempty"`
}

type ServiceConfig struct {
	Name      string           `json:"name"`
	Addr      string           `yaml:",omitempty" json:"addr,omitempty"`
	Bypass    string           `yaml:",omitempty" json:"bypass,omitempty"`
	Resolver  string           `yaml:",omitempty" json:"resolver,omitempty"`
	Hosts     string           `yaml:",omitempty" json:"hosts,omitempty"`
	Handler   *HandlerConfig   `yaml:",omitempty" json:"handler,omitempty"`
	Listener  *ListenerConfig  `yaml:",omitempty" json:"listener,omitempty"`
	Forwarder *ForwarderConfig `yaml:",omitempty" json:"forwarder,omitempty"`
}

type ChainConfig struct {
	Name     string          `json:"name"`
	Selector *SelectorConfig `yaml:",omitempty" json:"selector,omitempty"`
	Hops     []*HopConfig    `json:"hops"`
}

type HopConfig struct {
	Name     string          `json:"name"`
	Selector *SelectorConfig `yaml:",omitempty" json:"selector,omitempty"`
	Bypass   string          `yaml:",omitempty" json:"bypass,omitempty"`
	Resolver string          `yaml:",omitempty" json:"resolver,omitempty"`
	Hosts    string          `yaml:",omitempty" json:"hosts,omitempty"`
	Nodes    []*NodeConfig   `json:"nodes"`
}

type NodeConfig struct {
	Name      string           `json:"name"`
	Addr      string           `yaml:",omitempty" json:"addr,omitempty"`
	Bypass    string           `yaml:",omitempty" json:"bypass,omitempty"`
	Resolver  string           `yaml:",omitempty" json:"resolver,omitempty"`
	Hosts     string           `yaml:",omitempty" json:"hosts,omitempty"`
	Connector *ConnectorConfig `yaml:",omitempty" json:"connector,omitempty"`
	Dialer    *DialerConfig    `yaml:",omitempty" json:"dialer,omitempty"`
}

type Config struct {
	Services  []*ServiceConfig  `json:"services"`
	Chains    []*ChainConfig    `yaml:",omitempty" json:"chains,omitempty"`
	Bypasses  []*BypassConfig   `yaml:",omitempty" json:"bypasses,omitempty"`
	Resolvers []*ResolverConfig `yaml:",omitempty" json:"resolvers,omitempty"`
	Hosts     []*HostsConfig    `yaml:",omitempty" json:"hosts,omitempty"`
	TLS       *TLSConfig        `yaml:",omitempty" json:"tls,omitempty"`
	Log       *LogConfig        `yaml:",omitempty" json:"log,omitempty"`
	Profiling *ProfilingConfig  `yaml:",omitempty" json:"profiling,omitempty"`
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

func (c *Config) Write(w io.Writer, format string) error {
	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.Encode(c)
		return nil
	case "yaml":
		fallthrough
	default:
		enc := yaml.NewEncoder(w)
		defer enc.Close()

		return enc.Encode(c)
	}
}
