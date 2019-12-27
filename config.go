package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"

	"snet/proxy"
	"snet/proxy/http"
	"snet/proxy/ss"
	"snet/proxy/tls"
)

const (
	proxyScopeBypassCN = "bypassCN"
	proxyScopeGlobal   = "global"
)

const (
	DefaultLHost            = "127.0.0.1"
	DefaultLPort            = 1111
	DefaultProxyTimeout     = 30
	DefaultProxyType        = "ss"
	DefaultProxyScope       = proxyScopeBypassCN
	DefaultCNDNS            = "223.6.6.6"
	DefaultFQDNS            = "8.8.8.8"
	DefaultMode             = "local"
	DefaultPrefetchCount    = 10
	DefaultPrefetchInterval = 10
)

type Config struct {
	LHost                 string            `json:"listen-host"`
	LPort                 int               `json:"listen-port"`
	ProxyType             string            `json:"proxy-type"`
	ProxyTimeout          int               `json:"proxy-timeout"`
	ProxyScope            string            `json:"proxy-scope"`
	BypassHosts           []string          `json:"bypass-hosts"`
	BypassSrcIPs          []string          `json:"bypass-src-ips"`
	HTTPProxyHost         string            `json:"http-proxy-host"`
	HTTPProxyPort         int               `json:"http-proxy-port"`
	HTTPProxyAuthUser     string            `json:"http-proxy-auth-user"`
	HTTPProxyAuthPassword string            `json:"http-proxy-auth-password"`
	SSHost                string            `json:"ss-host"`
	SSPort                int               `json:"ss-port"`
	SSCphierMethod        string            `json:"ss-chpier-method"`
	SSPasswd              string            `json:"ss-passwd"`
	TLSHost               string            `json:"tls-host"`
	TLSPort               int               `json:"tls-port"`
	TLSToken              string            `json:"tls-token"`
	CNDNS                 string            `json:"cn-dns"`
	FQDNS                 string            `json:"fq-dns"`
	EnableDNSCache        bool              `json:"enable-dns-cache"`
	EnforceTTL            uint32            `json:"enforce-ttl"`
	DNSPrefetchEnable     bool              `json:"dns-prefetch-enable"`
	DNSPrefetchCount      int               `json:"dns-prefetch-count"`
	DNSPrefetchInterval   int               `json:"dns-prefetch-interval"`
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
	if config.DNSPrefetchCount == 0 {
		config.DNSPrefetchCount = DefaultPrefetchCount
	}
	if config.DNSPrefetchInterval == 0 {
		config.DNSPrefetchInterval = DefaultPrefetchInterval
	}
	return &config, nil
}

func genConfigByType(c *Config, proxyType string) (proxy.Config, error) {
	switch proxyType {
	case "ss":
		ip, err := resolvHostIP(c.SSHost)
		if err != nil {
			return nil, err
		}
		return &ss.Config{Host: ip, Port: c.SSPort, CipherMethod: c.SSCphierMethod, Password: c.SSPasswd}, nil
	case "http":
		ip, err := resolvHostIP(c.HTTPProxyHost)
		if err != nil {
			return nil, err
		}
		return &http.Config{Host: ip, Port: c.HTTPProxyPort, AuthUser: c.HTTPProxyAuthUser, AuthPassword: c.HTTPProxyAuthPassword}, nil
	case "tls":
		ip, err := resolvHostIP(c.TLSHost)
		if err != nil {
			return nil, err
		}
		return &tls.Config{Host: ip, Port: c.TLSPort, Token: c.TLSToken}, nil
	}
	return nil, nil
}

func resolvHostIP(host string) (net.IP, error) {
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, errors.New("No ip found for " + host)
	}
	return ips[0], nil
}
