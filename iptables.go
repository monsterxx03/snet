// ipset BYPASS_SNET:
// reversed ips + china ips + upstream ss ip + cn dns ip
//
// iptables -t nat -N SNET
// iptables -t nat -A SNET -p tcp -m set --match-set BYPASS_SNET dst -j RETURN
// iptables -t nat -A SNET -p tcp -j REDIRECT --to-ports 1111
// iptables -t nat -A OUTPUT -p tcp -j SNET
// For local
// iptables -t nat -A SNET -d 114.114.114.114 -j RETURN
// iptables -t nat -A SNET -p udp --dport 53 -j DNAT --to-destination 127.0.0.1:1111
// iptables -t nat -A OUTPUT -p udp --dport 53 -j SNET
// For router
// iptables -t nat -I PREROUTING -p tcp -j SNET
// iptables -t nat -I PREROUTING -p udp --dport 53 -j REDIRECT --to-port 1111
//
// For UDP
// iptables -t mangle -N SNET
// ip route add local default dev lo table 100
// ip rule add fwmark 0x01/0x01 lookup 100
// iptables -t mangle -A SNET --dport 53 -j RETURN
// iptables -t mangle -A SNET -d 0.0.0.0/8 -j RETURN
// iptables -t mangle -A SNET -d 10.0.0.0/8 -j RETURN
// iptables -t mangle -A SNET -d 127.0.0.0/8 -j RETURN
// iptables -t mangle -A SNET -d 169.254.0.0/16 -j RETURN
// iptables -t mangle -A SNET -d 172.16.0.0/12 -j RETURN
// iptables -t mangle -A SNET -d 192.168.0.0/16 -j RETURN
// iptables -t mangle -A SNET -d 224.0.0.0/4 -j RETURN
// iptables -t mangle -A SNET -d 240.0.0.0/4 -j RETURN
// iptables -t mangle -A SNET -p udp -j TPROXY --on-port 1111 --on-ip 127.0.0.1 --tproxy-mark 0x01/0x01
// iptables -t mangle -A PREROUTING -p udp -j SNET

package main

import (
	"fmt"
	"strconv"
)

const (
	SnetChainName = "SNET"
	ModeLocal     = "local"
	ModeRouter    = "router"
)

func setupIptableRules(mode string, snetHost string, snetPort int, dnsPort int, cnDNS string, bypassSetName string) error {
	cleanIptableRules(mode, snetHost, snetPort, dnsPort, bypassSetName)
	port := strconv.Itoa(snetPort)
	dport := strconv.Itoa(dnsPort)
	Sh("iptables -t nat -N", SnetChainName)
	Sh("iptables -t nat -A ", SnetChainName, "-p tcp -m set --match-set", bypassSetName, "dst -j RETURN")
	Sh("iptables -t nat -A ", SnetChainName, "-p tcp -j REDIRECT --to-ports", port)
	Sh("iptables -t nat -A OUTPUT -p tcp -j", SnetChainName)
	if mode == ModeLocal {
		// avoid outgoing cn dns query be redirected to snet, it's a loop!
		Sh("iptables -t nat -A", SnetChainName, "-d", cnDNS, "-j RETURN")
		// redirect all outgoing dns query to snet(except cn dns)
		Sh("iptables -t nat -A", SnetChainName, "-p udp --dport 53 -j DNAT --to-destination", snetHost+":"+dport)

		Sh("iptables -t nat -A OUTPUT -p udp --dport 53 -j", SnetChainName)
	}
	if mode == ModeRouter {
		Sh("iptables -t nat -I PREROUTING -p tcp -j", SnetChainName)
		Sh("iptables -t nat -I PREROUTING -p udp --dport 53 -j REDIRECT --to-port", dport)
	}
	return nil
}

func cleanIptableRules(mode string, snetHost string, snetPort int, dnsPort int, bypassSetName string) error {
	if mode != ModeLocal && mode != ModeRouter {
		return fmt.Errorf("Invalid mode %s", mode)
	}
	dport := strconv.Itoa(dnsPort)
	Sh("iptables -t nat -D OUTPUT -p tcp -j ", SnetChainName)
	if mode == ModeLocal {
		Sh("iptables -t nat -D", SnetChainName, "-p  udp --dport 53 -j DNAT --to-destination", snetHost+":"+dport)
		Sh("iptables -t nat -D OUTPUT -p udp --dport 53 -j", SnetChainName)
	}
	if mode == ModeRouter {
		Sh("iptables -t nat -D PREROUTING -p tcp -j", SnetChainName)
		Sh("iptables -t nat -D PREROUTING -p udp --dport 53 -j REDIRECT --to-port", dport)
	}
	Sh("iptables -t nat -F", SnetChainName)
	Sh("iptables -t nat -X", SnetChainName)
	return nil
}
