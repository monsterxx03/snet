package main

import (
	"encoding/json"
	"io/ioutil"
)

type Config struct {
	LHost          string `json:"listen-host"`
	LPort          int    `json:"listen-port"`
	SSHost         string `json:"ss-host"`
	SSPort         int    `json:"ss-port"`
	SSCphierMethod string `json:"ss-chpier-method"`
	SSPasswd       string `json:"ss-passwd"`
	CNDNS          string `json:"cn-dns"`
	FQDNS          string `json:"fq-dns"`
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
