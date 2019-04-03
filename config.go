package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"snet/proxy"
	ss "snet/proxy/ss"
)

type Config struct {
	LHost          string `json:"listen-host"`
	LPort          int    `json:"listen-port"`
	ProxyType      string `json:"proxy-type"`
	HttpProxyHost  string `json:"http-proxy-host"`
	HttpProxyPort  string `json:"http-proxy-port"`
	HttpProxyAuth  string `json:"http-proxy-auth"`
	SSHost         string `json:"ss-host"`
	SSPort         int    `json:"ss-port"`
	SSCphierMethod string `json:"ss-chpier-method"`
	SSPasswd       string `json:"ss-passwd"`
	CNDNS          string `json:"cn-dns"`
	FQDNS          string `json:"fq-dns"`
	EnableDNSCache bool   `json:"enable-dns-cache"`
	Mode           string `json:"mode"`
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
		return nil, errors.New("proxy-type required")
	}
	return &config, nil
}

func genConfigByType(c *Config, proxyType string) proxy.Config {
	switch proxyType {
	case "ss":
		return &ss.Config{Host: c.SSHost, Port: c.SSPort, CipherMethod: c.SSCphierMethod, Password: c.SSPasswd}
	}
	return nil
}
