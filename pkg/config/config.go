package config

import (
	"encoding/json"
	"io"
	"sync"
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

var (
	global    = &Config{}
	globalMux sync.RWMutex
)

func Global() *Config {
	globalMux.RLock()
	defer globalMux.RUnlock()

	cfg := &Config{}
	*cfg = *global
	return cfg
}

func SetGlobal(c *Config) {
	globalMux.Lock()
	defer globalMux.Unlock()

	global = c
}

type LogConfig struct {
	Output string `yaml:",omitempty" json:"output,omitempty"`
	Level  string `yaml:",omitempty" json:"level,omitempty"`
	Format string `yaml:",omitempty" json:"format,omitempty"`
}

type ProfilingConfig struct {
	Addr   string `json:"addr"`
	Enable bool   `json:"enable"`
}

type APIConfig struct {
	Addr       string      `json:"addr"`
	PathPrefix string      `yaml:"pathPrefix,omitempty" json:"pathPrefix,omitempty"`
	AccessLog  bool        `yaml:"accesslog,omitempty" json:"accesslog,omitempty"`
	Auth       *AuthConfig `yaml:",omitempty" json:"auth,omitempty"`
	Auther     string      `yaml:",omitempty" json:"auther,omitempty"`
}

type MetricsConfig struct {
	Enable bool   `json:"enable"`
	Addr   string `json:"addr"`
	Path   string `json:"path"`
}

type TLSConfig struct {
	CertFile   string `yaml:"certFile,omitempty" json:"certFile,omitempty"`
	KeyFile    string `yaml:"keyFile,omitempty" json:"keyFile,omitempty"`
	CAFile     string `yaml:"caFile,omitempty" json:"caFile,omitempty"`
	Secure     bool   `yaml:",omitempty" json:"secure,omitempty"`
	ServerName string `yaml:"serverName,omitempty" json:"serverName,omitempty"`
}

type AutherConfig struct {
	Name string `json:"name"`
	// inline, file, redis, etc.
	Type  string        `yaml:",omitempty" json:"type,omitempty"`
	Auths []*AuthConfig `yaml:",omitempty" json:"auths"`
	// File string        `yaml:",omitempty" json:"file"`
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

type AdmissionConfig struct {
	Name string `json:"name"`
	// inline, file, etc.
	Type     string   `yaml:",omitempty" json:"type,omitempty"`
	Reverse  bool     `yaml:",omitempty" json:"reverse,omitempty"`
	Matchers []string `json:"matchers"`
}

type BypassConfig struct {
	Name string `json:"name"`
	// inline, file, etc.
	Type     string   `yaml:",omitempty" json:"type,omitempty"`
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
	Name string `json:"name"`
	// inline, file, etc.
	Type        string              `yaml:",omitempty" json:"type,omitempty"`
	Nameservers []*NameserverConfig `json:"nameservers"`
}

type HostMappingConfig struct {
	IP       string   `json:"ip"`
	Hostname string   `json:"hostname"`
	Aliases  []string `yaml:",omitempty" json:"aliases,omitempty"`
}

type HostsConfig struct {
	Name string `json:"name"`
	// inline, file, etc.
	Type     string               `yaml:",omitempty" json:"type,omitempty"`
	Mappings []*HostMappingConfig `json:"mappings"`
}

type ListenerConfig struct {
	Type     string         `json:"type"`
	Chain    string         `yaml:",omitempty" json:"chain,omitempty"`
	Auther   string         `yaml:",omitempty" json:"auther,omitempty"`
	Auth     *AuthConfig    `yaml:",omitempty" json:"auth,omitempty"`
	TLS      *TLSConfig     `yaml:",omitempty" json:"tls,omitempty"`
	Metadata map[string]any `yaml:",omitempty" json:"metadata,omitempty"`
}

type HandlerConfig struct {
	Type     string         `json:"type"`
	Retries  int            `yaml:",omitempty" json:"retries,omitempty"`
	Chain    string         `yaml:",omitempty" json:"chain,omitempty"`
	Auther   string         `yaml:",omitempty" json:"auther,omitempty"`
	Auth     *AuthConfig    `yaml:",omitempty" json:"auth,omitempty"`
	TLS      *TLSConfig     `yaml:",omitempty" json:"tls,omitempty"`
	Metadata map[string]any `yaml:",omitempty" json:"metadata,omitempty"`
}

type ForwarderConfig struct {
	Targets  []string        `json:"targets"`
	Selector *SelectorConfig `yaml:",omitempty" json:"selector,omitempty"`
}

type DialerConfig struct {
	Type     string         `json:"type"`
	Auth     *AuthConfig    `yaml:",omitempty" json:"auth,omitempty"`
	TLS      *TLSConfig     `yaml:",omitempty" json:"tls,omitempty"`
	Metadata map[string]any `yaml:",omitempty" json:"metadata,omitempty"`
}

type ConnectorConfig struct {
	Type     string         `json:"type"`
	Auth     *AuthConfig    `yaml:",omitempty" json:"auth,omitempty"`
	TLS      *TLSConfig     `yaml:",omitempty" json:"tls,omitempty"`
	Metadata map[string]any `yaml:",omitempty" json:"metadata,omitempty"`
}

type ServiceConfig struct {
	Name      string           `json:"name"`
	Addr      string           `yaml:",omitempty" json:"addr,omitempty"`
	Interface string           `yaml:",omitempty" json:"interface,omitempty"`
	Admission string           `yaml:",omitempty" json:"admission,omitempty"`
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
	Name      string          `json:"name"`
	Interface string          `yaml:",omitempty" json:"interface,omitempty"`
	Selector  *SelectorConfig `yaml:",omitempty" json:"selector,omitempty"`
	Bypass    string          `yaml:",omitempty" json:"bypass,omitempty"`
	Resolver  string          `yaml:",omitempty" json:"resolver,omitempty"`
	Hosts     string          `yaml:",omitempty" json:"hosts,omitempty"`
	Nodes     []*NodeConfig   `json:"nodes"`
}

type NodeConfig struct {
	Name      string           `json:"name"`
	Addr      string           `yaml:",omitempty" json:"addr,omitempty"`
	Interface string           `yaml:",omitempty" json:"interface,omitempty"`
	Bypass    string           `yaml:",omitempty" json:"bypass,omitempty"`
	Resolver  string           `yaml:",omitempty" json:"resolver,omitempty"`
	Hosts     string           `yaml:",omitempty" json:"hosts,omitempty"`
	Connector *ConnectorConfig `yaml:",omitempty" json:"connector,omitempty"`
	Dialer    *DialerConfig    `yaml:",omitempty" json:"dialer,omitempty"`
}

type Config struct {
	Services   []*ServiceConfig   `json:"services"`
	Chains     []*ChainConfig     `yaml:",omitempty" json:"chains,omitempty"`
	Authers    []*AutherConfig    `yaml:",omitempty" json:"authers,omitempty"`
	Admissions []*AdmissionConfig `yaml:",omitempty" json:"admissions,omitempty"`
	Bypasses   []*BypassConfig    `yaml:",omitempty" json:"bypasses,omitempty"`
	Resolvers  []*ResolverConfig  `yaml:",omitempty" json:"resolvers,omitempty"`
	Hosts      []*HostsConfig     `yaml:",omitempty" json:"hosts,omitempty"`
	TLS        *TLSConfig         `yaml:",omitempty" json:"tls,omitempty"`
	Log        *LogConfig         `yaml:",omitempty" json:"log,omitempty"`
	Profiling  *ProfilingConfig   `yaml:",omitempty" json:"profiling,omitempty"`
	API        *APIConfig         `yaml:",omitempty" json:"api,omitempty"`
	Metrics    *MetricsConfig     `yaml:",omitempty" json:"metrics,omitempty"`
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
