package main

import (
	"strconv"
)

func setBypassRule(setName string) error {
	// only bypass tcp now, since default dns server is 192.168.1.1 for systemd-resolver, which will be byassed
	if _, err := Sh("iptables -t nat -A OUTPUT -p tcp -m set --match-set", setName, "dst -j RETURN"); err != nil {
		return err
	}
	return nil
}

func delByassRule(setName string) error {
	if _, err := Sh("iptables -t nat -D OUTPUT -p tcp -m set --match-set", setName, "dst -j RETURN"); err != nil {
		return err
	}
	return nil
}

func setRedirectRule(tgtPort int) error {
	if _, err := Sh("iptables -t nat -A OUTPUT -p tcp -j REDIRECT --to-ports", strconv.Itoa(tgtPort)); err != nil {
		return err
	}
	return nil
}

func delRedirectRule(tgtPort int) error {
	if _, err := Sh("iptables -t nat -D OUTPUT -p tcp -j REDIRECT --to-ports", strconv.Itoa(tgtPort)); err != nil {
		return nil
	}
	return nil
}

// not used now, since some system will query dns via ipv6
func setDNSRule(listen string) error {
	if _, err := Sh("iptables -t nat -A OUTPUT -p udp --dport 53 -j DNAT --to-destination", listen); err != nil {
		return err
	}
	return nil
}

func delDNSRule(listen string) error {
	if _, err := Sh("iptables -t nat -D OUTPUT -p udp --dport 53 -j DNAT --to-destination", listen); err != nil {
		return err
	}
	return nil
}
