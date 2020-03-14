package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"

	"snet/proxy"
	"snet/proxy/http"
	"snet/proxy/socks5"
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
	AsUpstream              bool              `json:"as-upstream"`
	LHost                   string            `json:"listen-host"`
	LPort                   int               `json:"listen-port"`
	ProxyType               string            `json:"proxy-type"`
	ProxyTimeout            int               `json:"proxy-timeout"`
	ProxyScope              string            `json:"proxy-scope"`
	BypassHosts             []string          `json:"bypass-hosts"`
	BypassSrcIPs            []string          `json:"bypass-src-ips"`
	HTTPProxyHost           string            `json:"http-proxy-host"`
	HTTPProxyPort           int               `json:"http-proxy-port"`
	HTTPProxyAuthUser       string            `json:"http-proxy-auth-user"`
	HTTPProxyAuthPassword   string            `json:"http-proxy-auth-password"`
	SSHost                  string            `json:"ss-host"`
	SSPort                  int               `json:"ss-port"`
	SSCphierMethod          string            `json:"ss-chpier-method"`
	SSPasswd                string            `json:"ss-passwd"`
	TLSHost                 string            `json:"tls-host"`
	TLSPort                 int               `json:"tls-port"`
	TLSToken                string            `json:"tls-token"`
	SOCKS5Host              string            `json:"socks5-host"`
	SOCKS5Port              int               `json:"socks5-port"`
	SOCKS5AuthUser          string            `json:"socks5-auth-user"`
	SOCKS5AuthPassword      string            `json:"socks5-auth-password"`
	CNDNS                   string            `json:"cn-dns"`
	FQDNS                   string            `json:"fq-dns"`
	EnableDNSCache          bool              `json:"enable-dns-cache"`
	EnforceTTL              uint32            `json:"enforce-ttl"`
	DNSPrefetchEnable       bool              `json:"dns-prefetch-enable"`
	DNSPrefetchCount        int               `json:"dns-prefetch-count"`
	DNSPrefetchInterval     int               `json:"dns-prefetch-interval"`
	DisableQTypes           []string          `json:"disable-qtypes"`
	ForceFQ                 []string          `json:"force-fq"`
	HostMap                 map[string]string `json:"host-map"`
	BlockHostFile           string            `json:"block-host-file"`
	BlockHosts              []string          `json:"block-hosts"`
	Mode                    string            `json:"mode"`
	UpstreamType            string            `json:"upstream-type"`
	UpstreamTLSServerListen string            `json:"upstream-tls-server-listen"`
	UpstreamTLSKey          string            `json:"upstream-tls-key"`
	UpstreamTLSCRT          string            `json:"upstream-tls-crt"`
	UpstreamTLSToken        string            `json:"upstream-tls-token"`
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
	case "socks5":
		ip, err := resolvHostIP(c.SOCKS5Host)
		if err != nil {
			return nil, err
		}
		return &socks5.Config{Host: ip, Port: c.SOCKS5Port, AuthUser: c.SOCKS5AuthUser, AuthPassword: c.SOCKS5AuthPassword}, nil
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
