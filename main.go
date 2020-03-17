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

	"snet/config"
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

func runClient(c *config.Config) {
	dnsPort := c.LPort + 100
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

	cleanupCallback := func() {
		l.Info("cleanup redirector rules")
		redir.CleanupRules(c.Mode, c.LHost, c.LPort, dnsPort)
		redir.Destroy()
	}

	if *clean {
		cleanupCallback()
		os.Exit(0)
	}
	s, err := NewServer(c)
	exitOnError(err, nil)
	proxyIP := s.proxy.GetProxyIP()
	exitOnError(err, nil)

	exitOnError(redir.Init(), nil)
	exitOnError(redir.ByPass(proxyIP.String()), nil)

	exitOnError(redir.SetupRules(c.Mode, c.LHost, c.LPort, dnsPort, c.CNDNS), cleanupCallback)

	dns, err := dns.NewServer(c, dnsPort, Chnroutes, l)
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

func runTLSServer(c *config.Config) {
	if c.UpstreamTLSToken == "" {
		exitOnError(errors.New("missing upstream-tls-token"), nil)
	}
	cert, err := tls.LoadX509KeyPair(c.UpstreamTLSCRT, c.UpstreamTLSKey)
	exitOnError(err, nil)
	tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}}
	ln, err := tls.Listen("tcp", c.UpstreamTLSServerListen, tlsCfg)
	exitOnError(err, nil)
	l.Info("TLS server running:", c.UpstreamTLSServerListen)
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
			if string(b) != c.UpstreamTLSToken {
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
	c, err := config.LoadConfig(*configFile)
	exitOnError(err, nil)
	if c.AsUpstream {
		switch c.UpstreamType {
		case "tls":
			runTLSServer(c)
		default:
			panic("unknow upstream-type:" + c.UpstreamType)
		}
	} else {
		runClient(c)
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
