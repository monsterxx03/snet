package main

import (
	"strconv"
)

const (
	SnetChainName = "SNET"
)

type SNETChain struct {
	Name string
}

func NewSNETChain() *SNETChain {
	return &SNETChain{
		Name: SnetChainName,
	}
}

// Init setup some a SNET chain, it will capture all tcp & dns traffic.
func (t *SNETChain) Init() error {
	if _, err := Sh("iptables -t nat -N", t.Name); err != nil {
		LOG.Err("Failed to create iptable chain", t.Name)
		return err
	}
	// let snet chain handle all tcp traffic
	if _, err := Sh("iptables -t nat -A OUTPUT -p tcp -j", t.Name); err != nil {
		return err
	}
	// let snet chain handle all udp dns query
	if _, err := Sh("iptables -t nat -A OUTPUT -p udp --dport 53 -j", t.Name); err != nil {
		return err
	}
	return nil
}

func (t *SNETChain) addRule(rule string) error {
	if _, err := Sh("iptables -t nat -A", t.Name, rule); err != nil {
		return err
	}
	return nil
}

// BypassIPSet will set rule to bypass tcp redirect for some ip ranges.
func (t *SNETChain) BypassIPSet(set *IPSet) error {
	if err := t.addRule("-p tcp -m set --match-set " + set.Name + " dst -j RETURN"); err != nil {
		return err
	}
	return nil
}

// RedirectTCP will redirect all out going tcp traffic to snet's tcp port
func (t *SNETChain) RedirectTCP(tgtPort int) error {
	if err := t.addRule("-p tcp -j REDIRECT --to-ports " + strconv.Itoa(tgtPort)); err != nil {
		return err
	}
	return nil
}

// RedirectDNS will redirect all udp dns query to snet's udp port, but bypass cn dns
func (t *SNETChain) RedirectDNS(localListenAddr, cnDNS string) error {
	// bypass cn dns, otherwise will dns query will be in loop
	if err := t.addRule("-d " + cnDNS + " -j RETURN"); err != nil {
		return err
	}
	// redirect output dns query to snet's udp port
	if err := t.addRule("-p udp --dport 53 -j DNAT --to-destination " + localListenAddr); err != nil {
		return err
	}
	return nil
}

func (t *SNETChain) Destroy() {
	Sh("iptables -t nat -D OUTPUT -p tcp -j", t.Name)
	Sh("iptables -t nat -D OUTPUT -p udp --dport 53 -j", t.Name)
	Sh("iptables -t nat -F", t.Name)
	Sh("iptables -t nat -X", t.Name)
}
