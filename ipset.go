package main

import (
	"log"
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
	if out, err := Sh("which ipset"); err != nil {
		log.Println("ipset not found", out)
		return nil, err
	}
	return &IPSet{Name: setName, bypassCidrs: whitelistCIDR}, nil
}

// Init will create a bypass ipset, and config iptables to RETURN traffic in this set.
func (s *IPSet) Init() error {
	Sh("ipset destroy", s.Name)
	if out, err := Sh("ipset create", s.Name, "hash:net"); err != nil {
		log.Println(out)
		return err
	}
	for _, cidr := range s.bypassCidrs {
		if out, err := Sh("ipset add", s.Name, cidr); err != nil {
			log.Println(out)
			return err
		}
	}
	return nil
}

func (s *IPSet) Bypass(ip string) error {
	s.bypassCidrs = append(s.bypassCidrs, ip)
	if out, err := Sh("ipset add", s.Name, ip); err != nil {
		log.Println(out)
		return err
	}
	return nil
}

func (s *IPSet) Destroy() error {
	if out, err := Sh("ipset destroy", s.Name); err != nil {
		log.Println(out)
		return err
	}
	return nil
}
