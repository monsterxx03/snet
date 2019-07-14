package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"snet/proxy"
	http "snet/proxy/http"
	ss "snet/proxy/ss"
	"time"
)

const (
	proxyScopeBypassCN = "bypassCN"
	proxyScopeGlobal   = "global"
)

const (
	DefaultLHost        = "127.0.0.1"
	DefaultLPort        = 1111
	DefaultProxyTimeout = 5
	DefaultProxyType    = "ss"
	DefaultProxyScope   = proxyScopeBypassCN
	DefaultCNDNS        = "223.6.6.6"
	DefaultFQDNS        = "8.8.8.8"
	DefaultMode         = "local"
)

type Config struct {
	LHost                 string            `json:"listen-host"`
	LPort                 int               `json:"listen-port"`
	ProxyType             string            `json:"proxy-type"`
	ProxyTimeout          int               `json:"proxy-timeout"`
	ProxyScope            string            `json:"proxy-scope"`
	HttpProxyHost         string            `json:"http-proxy-host"`
	HttpProxyPort         int               `json:"http-proxy-port"`
	HttpProxyAuthUser     string            `json:"http-proxy-auth-user"`
	HttpProxyAuthPassword string            `json:"http-proxy-auth-password"`
	SSHost                string            `json:"ss-host"`
	SSPort                int               `json:"ss-port"`
	SSCphierMethod        string            `json:"ss-chpier-method"`
	SSPasswd              string            `json:"ss-passwd"`
	CNDNS                 string            `json:"cn-dns"`
	FQDNS                 string            `json:"fq-dns"`
	EnableDNSCache        bool              `json:"enable-dns-cache"`
	EnforceTTL            uint32            `json:"enforce-ttl"`
	DisableQTypes         []string          `json:"disable-qtypes"`
	ForceFQ               []string          `json:"force-fq"`
	HostMap               map[string]string `json:"host-map"`
	BlockHostFile         string            `json:"block-host-file"`
	BlockHosts            []string          `json:"block-hosts"`
	Mode                  string            `json:"mode"`
}

func LoadConfig(configPath string) (*Config, error) {
	var config Config
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	if config.ProxyType == "" {
		return nil, errors.New("missing proxy-type")
	}
	switch config.ProxyScope {
	case "":
		config.ProxyScope = proxyScopeBypassCN
	case proxyScopeGlobal, proxyScopeBypassCN:
	default:
		return nil, errors.New("invalid proxy-scope " + config.ProxyScope)
	}
	if config.ProxyScope == "" {
		config.ProxyScope = DefaultProxyScope
	}
	if config.LHost == "" {
		config.LHost = DefaultLHost
	}
	if config.LPort == 0 {
		config.LPort = DefaultLPort
	}
	if config.ProxyTimeout == 0 {
		config.ProxyTimeout = DefaultProxyTimeout
	}
	if config.CNDNS == "" {
		config.CNDNS = DefaultCNDNS
	}
	if config.FQDNS == "" {
		config.FQDNS = DefaultFQDNS
	}
	if config.Mode == "" {
		config.Mode = DefaultMode
	}
	return &config, nil
}

func genConfigByType(c *Config, proxyType string) proxy.Config {
	switch proxyType {
	case "ss":
		return &ss.Config{Host: c.SSHost, Port: c.SSPort, CipherMethod: c.SSCphierMethod, Password: c.SSPasswd, Timeout: time.Second * time.Duration(c.ProxyTimeout)}
	case "http":
		return &http.Config{Host: c.HttpProxyHost, Port: c.HttpProxyPort, AuthUser: c.HttpProxyAuthUser, AuthPassword: c.HttpProxyAuthPassword, Timeout: time.Second * time.Duration(c.ProxyTimeout)}
	}
	return nil
}
