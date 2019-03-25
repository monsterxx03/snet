package main

import (
	"net"
)

type IPChecker struct {
	cnCidrs []*net.IPNet
}

func NewIPChecker() (*IPChecker, error) {
	cidrs := make([]*net.IPNet, 0, len(Chnroutes))
	// init cn cidrs
	for _, route := range Chnroutes {
		_, ipnet, err := net.ParseCIDR(route)
		if err != nil {
			return nil, err
		}
		cidrs = append(cidrs, ipnet)
	}
	return &IPChecker{cidrs}, nil
}

// TODO speedup
func (c *IPChecker) InChina(ip net.IP) bool {
	for _, ipnet := range c.cnCidrs {
		if ipnet.Contains(ip) {
			return true
		}
	}
	return false
}
