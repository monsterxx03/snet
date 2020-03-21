package main

import (
	"context"
	"net"

	"snet/config"
	"snet/dns"
	"snet/redirector"
)

type LocalServer struct {
	cfg      *config.Config
	redir    redirector.Redirector
	dnServer *dns.DNS
	dnsPort  int
	server   *Server
	ctx      context.Context
}

func (s *LocalServer) Clean() {
	l.Info("cleanup redirector rules")
	s.redir.CleanupRules(s.cfg.Mode, s.cfg.LHost, s.cfg.LPort, s.dnsPort)
	s.redir.Destroy()
}

func (s *LocalServer) SetupRedirector() error {
	proxyIP := s.server.proxy.GetProxyIP()
	if err := s.redir.Init(); err != nil {
		return err
	}
	if err := s.redir.ByPass(proxyIP.String()); err != nil {
		return err
	}
	if err := s.redir.SetupRules(s.cfg.Mode, s.cfg.LHost, s.cfg.LPort, s.dnsPort, s.cfg.CNDNS); err != nil {
		s.Clean()
		return err
	}
	return nil
}

func (s *LocalServer) Run() {
	var err error

	s.server, err = NewServer(s.cfg)
	exitOnError(err, nil)
	exitOnError(s.SetupRedirector(), nil)

	go s.dnServer.Run()
	go s.server.Run()
	<-s.ctx.Done()
	s.dnServer.Shutdown()
	s.server.Shutdown()
	s.Clean()
}

func NewLocalServer(ctx context.Context, c *config.Config) *LocalServer {
	if c.ProxyType == "" {
		panic("missing proxy-type")
	}
	switch c.ProxyScope {
	case "":
		c.ProxyScope = proxyScopeBypassCN
	case proxyScopeGlobal, proxyScopeBypassCN:
	default:
		panic("invalid proxy-scope " + c.ProxyScope)
	}
	if c.ProxyScope == "" {
		c.ProxyScope = DefaultProxyScope
	}
	if c.LHost == "" {
		c.LHost = DefaultLHost
	}
	if c.LPort == 0 {
		c.LPort = DefaultLPort
	}
	if c.ProxyTimeout == 0 {
		c.ProxyTimeout = DefaultProxyTimeout
	}
	if c.CNDNS == "" {
		c.CNDNS = DefaultCNDNS
	}
	if c.FQDNS == "" {
		c.FQDNS = DefaultFQDNS
	}
	if c.Mode == "" {
		c.Mode = DefaultMode
	}
	if c.DNSPrefetchCount == 0 {
		c.DNSPrefetchCount = DefaultPrefetchCount
	}
	if c.DNSPrefetchInterval == 0 {
		c.DNSPrefetchInterval = DefaultPrefetchInterval
	}

	// bypass logic
	var bypassCidrs []string
	if c.ProxyScope == proxyScopeBypassCN {
		bypassCidrs = Chnroutes
	} else {
		bypassCidrs = []string{}
	}
	if !*clean {
		for _, h := range c.BypassHosts {
			ips, err := net.LookupIP(h)
			if err != nil {
				exitOnError(err, nil)
			}
			for _, ip := range ips {
				bypassCidrs = append(bypassCidrs, ip.String())
			}
		}

	}
	redir, err := redirector.NewRedirector(bypassCidrs, c.BypassSrcIPs, l)
	exitOnError(err, nil)
	dnsPort := c.LPort + 100
	dns, err := dns.NewServer(c, dnsPort, Chnroutes, l)
	exitOnError(err, nil)
	return &LocalServer{cfg: c, redir: redir, dnsPort: c.LPort + 100, dnServer: dns, ctx: ctx}
}
