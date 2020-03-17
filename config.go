package main

import (
	"errors"
	"net"

	"snet/config"
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

func genConfigByType(c *config.Config, proxyType string) (proxy.Config, error) {
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
