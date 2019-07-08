package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"snet/dns"
	"snet/logger"
	"snet/redirector"
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

var configFile = flag.String("config", "", "json cofig file path")
var clean = flag.Bool("clean", false, "cleanup iptables and ipset")
var version = flag.Bool("version", false, "print version only")
var verbose = flag.Bool("v", false, "verbose output")
var l *logger.Logger

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
	dnsPort := config.LPort + 100

	var bypassCidrs []string
	if config.ProxyScope == proxyScopeBypassCN {
		bypassCidrs = Chnroutes
	} else {
		bypassCidrs = []string{}
	}
	redir, err := redirector.NewRedirector(bypassCidrs, l)
	exitOnError(err, nil)

	cleanupCallback := func() {
		redir.CleanupRules(config.Mode, config.LHost, config.LPort, dnsPort)
		redir.Destroy()
	}

	if *clean {
		cleanupCallback()
		os.Exit(0)
	}
	s, err := NewServer(config)
	exitOnError(err, nil)
	errCh := make(chan error)
	proxyIP := s.proxy.GetProxyIP()
	exitOnError(err, nil)

	exitOnError(redir.Init(), nil)
	exitOnError(redir.ByPass(proxyIP.String()), nil)

	redir.SetupRules(config.Mode, config.LHost, config.LPort, dnsPort, config.CNDNS)
	exitOnError(err, cleanupCallback)

	addr := fmt.Sprintf("%s:%d", config.LHost, dnsPort)
	dns, err := dns.NewServer(addr, config.CNDNS, config.FQDNS, config.EnableDNSCache, config.EnforceTTL, config.DisableQTypes, config.ForceFQ, config.HostMap, config.BlockHostFile, Chnroutes, l)
	exitOnError(err, cleanupCallback)
	go func() {
		errCh <- dns.Run()
	}()

	go func() {
		errCh <- s.Run()
	}()
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGHUP)
		l.Info("Got signal:", <-c)
		errCh <- nil
	}()

	if err := <-errCh; err != nil {
		l.Info(err)
	}

	cleanupCallback()

	if err := dns.Shutdown(); err != nil {
		l.Warn("Error during shutdown dns server", err)
	}
	if err := s.Shutdown(); err != nil {
		l.Warn("Error during shutdown server", err)
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
