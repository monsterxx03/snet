package main

import (
	"context"
	"net"
	"sync"

	"snet/config"
	"snet/dns"
	"snet/redirector"
)

type LocalServer struct {
	cfg      *config.Config
	cfgChan  chan *config.Config
	redir    redirector.Redirector
	dnServer *dns.DNS
	dnsPort  int
	server   *Server
	quit     bool
	qlock    sync.Mutex
	ctx      context.Context
}

func (s *LocalServer) Clean() {
	l.Info("cleanup redirector rules")
	s.redir.CleanupRules(s.cfg.Mode, s.cfg.LHost, s.cfg.LPort, s.dnsPort)
	s.redir.Destroy()
}

func (s *LocalServer) SetupRedirector() error {
	// bypass logic
	var bypassCidrs []string
	var err error
	if s.cfg.ProxyScope == config.ProxyScopeBypassCN {
		bypassCidrs = Chnroutes
	} else {
		bypassCidrs = []string{}
	}
	for _, h := range s.cfg.BypassHosts {
		ips, err := net.LookupIP(h)
		if err != nil {
			exitOnError(err, nil)
		}
		for _, ip := range ips {
			bypassCidrs = append(bypassCidrs, ip.String())
		}
	}

	s.redir, err = redirector.NewRedirector(bypassCidrs, s.cfg.BypassSrcIPs, l)
	exitOnError(err, nil)
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

func (s *LocalServer) SetupDNServer() error {
	dnsPort := s.cfg.LPort + 100
	dns, err := dns.NewServer(s.cfg, dnsPort, Chnroutes, l)
	if err != nil {
		return err
	}
	s.dnsPort = dnsPort
	s.dnServer = dns
	return nil
}

func (s *LocalServer) Shutdown() {
	s.qlock.Lock()
	defer s.qlock.Unlock()
	if s.quit {
		return
	}
	s.dnServer.Shutdown()
	s.server.Shutdown()
	s.Clean()
	s.quit = true
}

func (s *LocalServer) Run() {
	var err error
	s.quit = false
	s.server, err = NewServer(s.cfg)
	exitOnError(err, nil)
	exitOnError(s.SetupDNServer(), nil)
	exitOnError(s.SetupRedirector(), nil)

	go func() {
		cfg := <-s.cfgChan
		s.Shutdown()
		s.cfg = cfg
		s.Run()
	}()

	go s.dnServer.Run()
	go s.server.Run()
	<-s.ctx.Done()
	s.Shutdown()
}

func NewLocalServer(ctx context.Context, c *config.Config) *LocalServer {
	return &LocalServer{cfg: c, cfgChan: make(chan *config.Config), ctx: ctx}
}
