package config

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"strings"
	"time"
)

var (
	defaultConfig = Config{
		ListenAddr: ":8080",
		Clusters:   []Cluster{defaultCluster},
	}

	defaultCluster = Cluster{
		Scheme:   "http",
		OutUsers: []OutUser{defaultOutUser},
	}

	defaultOutUser = OutUser{
		Name: "default",
	}
)

// Config is an structure to describe access and proxy rules
// The simplest configuration consists of:
// 	 cluster description - see <remote_servers> section in CH config.xml
// 	 and users - who allowed to access proxy
// Users requests are mapped to CH-cluster via `to_cluster` option
// with credentials of cluster user from `to_user` option
type Config struct {
	// TCP address to listen to for http
	// Default is `localhost:8080`
	ListenAddr string `yaml:"listen_addr,omitempty"`

	// TCP address to listen to for https
	ListenTLSAddr string `yaml:"listen_tls_addr,omitempty"`

	// Path to the directory where letsencrypt certs are cache
	CertCacheDir string `yaml:"cert_cache_dir,omitempty"`

	// Whether to print debug logs
	LogDebug bool `yaml:"log_debug,omitempty"`

	Clusters []Cluster `yaml:"clusters"`

	GlobalUsers []GlobalUser `yaml:"global_users"`

	// Catches all undefined fields
	XXX map[string]interface{} `yaml:",inline"`
}

// Validates passed configuration by additional marshalling
// to ensure that all rules and checks were applied
func (c *Config) Validate() error {
	content, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("error while marshalling config: %s", err)
	}

	cfg := &Config{}
	return yaml.Unmarshal([]byte(content), cfg)
	// TODO: check listen addr, consistency of global and out users
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = defaultConfig

	// set c to the defaults and then overwrite it with the input.
	type plain Config
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}

	if len(c.GlobalUsers) == 0 {
		return fmt.Errorf("field `global_users` must contain at least 1 user")
	}

	return checkOverflow(c.XXX, "config")
}

// Cluster is an structure to describe CH cluster configuration
// The simplest configuration consists of:
// 	 cluster description - see <remote_servers> section in CH config.xml
// 	 and users - see <users> section in CH users.xml
type Cluster struct {
	// Name of ClickHouse cluster
	Name string `yaml:"name"`

	// Scheme: `http` or `https`; would be applied to all nodes
	// default value is `http`
	Scheme string `yaml:"scheme,omitempty"`

	// Nodes - list of nodes addresses
	Nodes []string `yaml:"nodes"`

	// OutUsers - list of ClickHouse users
	OutUsers []OutUser `yaml:"out_users"`

	// Catches all undefined fields
	XXX map[string]interface{} `yaml:",inline"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (c *Cluster) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = defaultCluster

	type plain Cluster
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}

	// TODO: check if it is already checked by Unmarshall
	if len(c.Nodes) == 0 {
		return fmt.Errorf("field `nodes` must contain at least 1 address")
	}

	if c.Scheme != "http" && c.Scheme != "https" {
		return fmt.Errorf("field `scheme` must be `http` or `https`. Got %q instead", c.Scheme)
	}

	return checkOverflow(c.XXX, "cluster")
}

// GlobalUser struct describes list of allowed users
// which requests will be proxied to ClickHouse
type GlobalUser struct {
	// User name
	Name string `yaml:"name"`

	// User password to access proxy with basic auth
	Password string `yaml:"password,omitempty"`

	// ToCluster is the name of cluster where requests
	// will be proxied
	ToCluster string `yaml:"to_cluster"`

	// ToUser is the name of out_user from cluster ToCluster whom credentials
	// will be used for proxying request to CH
	ToUser string `yaml:"to_user"`

	// Maximum number of concurrently running queries for user
	// if omitted or zero - no limits would be applied
	MaxConcurrentQueries uint32 `yaml:"max_concurrent_queries,omitempty"`

	// Maximum duration of query execution for user
	// if omitted or zero - no limits would be applied
	MaxExecutionTime time.Duration `yaml:"max_execution_time,omitempty"`

	// List of networks that access is allowed from
	// Each list item could be IP address or subnet mask
	// if omitted or zero - no limits would be applied
	AllowedNetworks []string `yaml:"allowed_networks,omitempty"`

	// Catches all undefined fields
	XXX map[string]interface{} `yaml:",inline"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (u *GlobalUser) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain GlobalUser
	if err := unmarshal((*plain)(u)); err != nil {
		return err
	}

	return checkOverflow(u.XXX, "out_users")
}

// User struct describes simplest <users> configuration
type OutUser struct {
	// User name in ClickHouse users.xml config
	Name string `yaml:"name"`

	// User password in ClickHouse users.xml config
	Password string `yaml:"password,omitempty"`

	// Maximum number of concurrently running queries for user
	// if omitted or zero - no limits would be applied
	MaxConcurrentQueries uint32 `yaml:"max_concurrent_queries,omitempty"`

	// Maximum duration of query executing for user
	// if omitted or zero - no limits would be applied
	MaxExecutionTime time.Duration `yaml:"max_execution_time,omitempty"`

	// Catches all undefined fields
	XXX map[string]interface{} `yaml:",inline"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (u *OutUser) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain OutUser
	if err := unmarshal((*plain)(u)); err != nil {
		return err
	}

	return checkOverflow(u.XXX, "out_users")
}

// Loads and validates configuration from provided .yml file
func LoadFile(filename string) (*Config, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := yaml.Unmarshal([]byte(content), cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func checkOverflow(m map[string]interface{}, ctx string) error {
	if len(m) > 0 {
		var keys []string
		for k := range m {
			keys = append(keys, k)
		}
		return fmt.Errorf("unknown fields in %s: %s", ctx, strings.Join(keys, ", "))
	}
	return nil
}
