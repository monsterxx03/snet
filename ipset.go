package main

import (
	"io"
	exec "os/exec"
	"strings"
)

const setName = "BYPASS_SNET"

// https://en.wikipedia.org/wiki/Reserved_IP_addresses#Reserved_IPv4_addresses
var whitelistCIDR = []string{
	"0.0.0.0/8",
	"10.0.0.0/8",
	"100.64.0.0/10",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"172.16.0.0/12",
	"192.0.2.0/24",
	"192.88.99.0/24",
	"192.168.0.0/16",
	"192.18.0.0/15",
	"192.51.100.0/24",
	"203.0.113.0/24",
	"224.0.0.0/4",
	"240.0.0.0/4",
	"255.255.255.255/32",
}

type IPSet struct {
	Name        string
	bypassCidrs []string
}

func NewIPSet() (*IPSet, error) {
	if _, err := Sh("which ipset"); err != nil {
		return nil, err
	}
	bypass := append(Chnroutes, whitelistCIDR...)
	return &IPSet{Name: setName, bypassCidrs: bypass}, nil
}

// Init will create a bypass ipset, and config iptables to RETURN traffic in this set.
func (s *IPSet) Init() error {
	Sh("ipset destroy", s.Name)
	result := make([]string, 0, len(s.bypassCidrs)+1)
	result = append(result, "create "+s.Name+" hash:net family inet hashsize 1024 maxelem 65536")
	for _, route := range s.bypassCidrs {
		result = append(result, "add "+s.Name+" "+route)
	}
	cmd := exec.Command("ipset", "restore")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, strings.Join(result, "\n"))
	}()
	out, err := cmd.CombinedOutput()
	if err != nil {
		LOG.Err(out)
		return err
	}
	return nil
}

func (s *IPSet) Bypass(ip string) error {
	s.bypassCidrs = append(s.bypassCidrs, ip)
	if _, err := Sh("ipset add", s.Name, ip); err != nil {
		return err
	}
	return nil
}

func (s *IPSet) Destroy() error {
	if _, err := Sh("ipset destroy", s.Name); err != nil {
		return err
	}
	return nil
}
