package main

import (
	"log"
	"strconv"
)

func setBypassRule(setName string) error {
	if out, err := Sh("iptables -t nat -A OUTPUT -m set --match-set", setName, "dst -j RETURN"); err != nil {
		log.Println(out)
		return err
	}
	return nil
}


func setRedirectRule(tgtPort int) error {
	if out, err := Sh("iptables -t nat -A OUTPUT -p tcp -j REDIRECT --to-ports", strconv.Itoa(tgtPort)) ;err != nil {
		log.Println(out)
		return err
	}
	return nil
}

func setDNSRule(listen string) error {
	if out, err := Sh("iptables -t nat -A OUTPUT -p udp --dport 53 -j DNAT --to-destination", listen); err != nil {
		log.Println(out)
		return err
	}
	return nil
}

func delByassRule(setName string) error {
	if out, err := Sh("iptables -t nat -D OUTPUT -m set --match-set", setName, "dst -j RETURN"); err != nil {
		log.Println(out)
		return err
	}
	return nil
}

func delRedirectRule(tgtPort int) error {
	if out, err := Sh("iptables -t nat -D OUTPUT -p tcp -j REDIRECT --to-ports", strconv.Itoa(tgtPort)); err != nil {
		log.Println(out)
		return nil
	}
	return nil
}

func delDNSRule(listen string) error {
	if out, err := Sh("iptables -t nat -D OUTPUT -p udp --dport 53 -j DNAT --to-destination", listen); err != nil {
		log.Println(out)
		return err
	}
	return nil
}
