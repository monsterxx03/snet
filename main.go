package main

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"snet/dns"
	"snet/logger"
	"snet/redirector"
	"snet/utils"
)

//go:generate go run chnroutes_generate.go
//go:generate go run ad_hosts_generate.go

const (
	DefaultLogLevel = logger.INFO
)

var (
	sha1Ver string
	buildAt string
)

var configFile = flag.String("config", "", "json config file path, only used when working as client")
var clean = flag.Bool("clean", false, "cleanup iptables and ipset")
var version = flag.Bool("version", false, "print version only")
var verbose = flag.Bool("v", false, "verbose output")
var l *logger.Logger

func runClient(config *Config) {
	dnsPort := config.LPort + 100
	if config.ProxyType == "" {
		panic("missing proxy-type")
	}
	switch config.ProxyScope {
	case "":
		config.ProxyScope = proxyScopeBypassCN
	case proxyScopeGlobal, proxyScopeBypassCN:
	default:
		panic("invalid proxy-scope " + config.ProxyScope)
	}
	if config.ProxyScope == "" {
		config.ProxyScope = DefaultProxyScope
	}
	if config.LHost == "" {
		config.LHost = DefaultLHost
	}
	if config.LPort == 0 {
		config.LPort = DefaultLPort
	}
	if config.ProxyTimeout == 0 {
		config.ProxyTimeout = DefaultProxyTimeout
	}
	if config.CNDNS == "" {
		config.CNDNS = DefaultCNDNS
	}
	if config.FQDNS == "" {
		config.FQDNS = DefaultFQDNS
	}
	if config.Mode == "" {
		config.Mode = DefaultMode
	}
	if config.DNSPrefetchCount == 0 {
		config.DNSPrefetchCount = DefaultPrefetchCount
	}
	if config.DNSPrefetchInterval == 0 {
		config.DNSPrefetchInterval = DefaultPrefetchInterval
	}

	// bypass logic
	var bypassCidrs []string
	if config.ProxyScope == proxyScopeBypassCN {
		bypassCidrs = Chnroutes
	} else {
		bypassCidrs = []string{}
	}
	if !*clean {
		for _, h := range config.BypassHosts {
			ips, err := net.LookupIP(h)
			if err != nil {
				exitOnError(err, nil)
			}
			for _, ip := range ips {
				bypassCidrs = append(bypassCidrs, ip.String())
			}
		}

	}
	redir, err := redirector.NewRedirector(bypassCidrs, config.BypassSrcIPs, l)
	exitOnError(err, nil)

	cleanupCallback := func() {
		l.Info("cleanup redirector rules")
		redir.CleanupRules(config.Mode, config.LHost, config.LPort, dnsPort)
		redir.Destroy()
	}

	if *clean {
		cleanupCallback()
		os.Exit(0)
	}
	s, err := NewServer(config)
	exitOnError(err, nil)
	proxyIP := s.proxy.GetProxyIP()
	exitOnError(err, nil)

	exitOnError(redir.Init(), nil)
	exitOnError(redir.ByPass(proxyIP.String()), nil)

	exitOnError(redir.SetupRules(config.Mode, config.LHost, config.LPort, dnsPort, config.CNDNS), cleanupCallback)

	addr := fmt.Sprintf("%s:%d", config.LHost, dnsPort)
	dns, err := dns.NewServer(addr, config.CNDNS, config.FQDNS, config.EnableDNSCache, config.EnforceTTL, config.DisableQTypes, config.ForceFQ, config.HostMap, config.BlockHostFile, config.BlockHosts,
		config.DNSPrefetchEnable, config.DNSPrefetchCount, config.DNSPrefetchInterval,
		Chnroutes, l)
	exitOnError(err, cleanupCallback)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGHUP)
		l.Info("Got signal:", <-c)
		cancel()
	}()
	go dns.Run()
	go s.Run()
	<-ctx.Done()
	dns.Shutdown()
	s.Shutdown()
	cleanupCallback()
}

func runTLSServer(config *Config) {
	if config.UpstreamTLSToken == "" {
		exitOnError(errors.New("missing upstream-tls-token"), nil)
	}
	cert, err := tls.LoadX509KeyPair(config.UpstreamTLSCRT, config.UpstreamTLSKey)
	exitOnError(err, nil)
	tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}}
	ln, err := tls.Listen("tcp", config.UpstreamTLSServerListen, tlsCfg)
	exitOnError(err, nil)
	l.Info("TLS server running:", config.UpstreamTLSServerListen)
	defer ln.Close()
	for {
		conn, err := ln.Accept()
		if err != nil {
			l.Error(err)
			continue
		}
		go func(conn net.Conn) {
			defer conn.Close()
			b := make([]byte, 2)
			if _, err := conn.Read(b); err != nil {
				l.Error(err)
				return
			}
			tlen := binary.BigEndian.Uint16(b)
			b = make([]byte, int(tlen))
			if _, err := conn.Read(b); err != nil {
				l.Error(err)
				return
			}
			if string(b) != config.UpstreamTLSToken {
				l.Error("invalid token", string(b))
				return
			}

			b = make([]byte, 2)
			if _, err := conn.Read(b); err != nil {
				l.Error(err)
				return
			}
			hlen := binary.BigEndian.Uint16(b)
			b = make([]byte, int(hlen))
			if _, err := conn.Read(b); err != nil {
				l.Error(err)
				return
			}
			host := string(b)
			b = make([]byte, 2)
			if _, err := conn.Read(b); err != nil {
				l.Error(err)
				return
			}
			port := int(binary.BigEndian.Uint16(b))
			dstConn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
			if err != nil {
				l.Error(err)
				return
			}
			defer dstConn.Close()
			if err := utils.Pipe(conn, dstConn, time.Duration(30)*time.Second); err != nil {
				l.Error(err)
			}
		}(conn)
	}
}

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("Git: %s\n", sha1Ver)
		fmt.Printf("Build at: %s\n", buildAt)
		fmt.Printf("Chnroutes updated at: %s\n", ChnroutesTS)
		os.Exit(0)
	}

	if *verbose {
		l = logger.NewLogger(logger.DEBUG)
	} else {
		l = logger.NewLogger(DefaultLogLevel)
	}
	var err error

	if *configFile == "" {
		fmt.Println("-config is required")
		os.Exit(1)
	}
	config, err := LoadConfig(*configFile)
	exitOnError(err, nil)
	if config.AsUpstream {
		switch config.UpstreamType {
		case "tls":
			runTLSServer(config)
		default:
			panic("unknow upstream-type:" + config.UpstreamType)
		}
	} else {
		runClient(config)
	}
}

func exitOnError(err error, cb func()) {
	if err != nil {
		if cb != nil {
			cb()
		}
		l.Fatal(err)
	}
}
