package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

//go:generate go run chnroutes_generate.go

const (
	DefaultLHost        = "127.0.0.1"
	DefaultLPort        = 1111
	DefaultProxyTimeout = 5
	DefaultProxyType    = "ss"
	DefaultCNDNS        = "223.6.6.6"
	DefaultFQDNS        = "8.8.8.8"
	DefaultMode         = "local"
)

var (
	sha1Ver string
	buildAt string
)

var configFile = flag.String("config", "", "json cofig file path")
var verbose = flag.Bool("v", false, "verbose logging")
var version = flag.Bool("version", false, "print version only")
var clean = flag.Bool("clean", false, "cleanup iptables and ipset")

var LOG *Logger

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("Git: %s\n", sha1Ver)
		fmt.Printf("Build at: %s\n", buildAt)
		fmt.Printf("Chnroutes updated at: %s\n", ChnroutesTS)
		os.Exit(0)
	}

	var logLevel LogLevel
	var err error

	if *verbose {
		logLevel = LOG_DEBUG
	} else {
		logLevel = LOG_INFO
	}
	LOG = NewLogger(logLevel)

	ipset, err := NewIPSet()
	exitOnError(err)

	if *configFile == "" {
		fmt.Println("-config is required")
		os.Exit(1)
	}
	config, err := LoadConfig(*configFile)
	exitOnError(err)
	dnsPort := config.LPort + 100

	if *clean {
		cleanIptableRules(config.Mode, config.LHost, config.LPort, dnsPort, setName)
		ipset.Destroy()
		os.Exit(0)
	}
	s, err := NewServer(config)
	exitOnError(err)
	errCh := make(chan error)

	exitOnError(ipset.Init())
	proxyIP := s.proxy.GetProxyIP()
	exitOnError(err)
	exitOnError(ipset.Bypass(proxyIP.String()))
	setupIptableRules(config.Mode, config.LHost, config.LPort, dnsPort, config.CNDNS, setName)

	addr := fmt.Sprintf("%s:%d", config.LHost, dnsPort)
	dns, err := NewDNS(addr, config.CNDNS, config.FQDNS, config.EnableDNSCache, config.EnforceTTL, config.DisableQTypes)
	exitOnError(err)
	go func() {
		errCh <- dns.Run()
	}()

	go func() {
		errCh <- s.Run()
	}()
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGHUP)
		LOG.Info("Got signal:", <-c)
		errCh <- nil
	}()

	if err := <-errCh; err != nil {
		LOG.Err(err)
	}
	cleanIptableRules(config.Mode, config.LHost, config.LPort, dnsPort, setName)
	ipset.Destroy()

	if err := dns.Shutdown(); err != nil {
		LOG.Err("Error during shutdown dns server", err)
	}
	if err := s.Shutdown(); err != nil {
		LOG.Err("Error during shutdown server", err)
	}
}

func exitOnError(err error) {
	if err != nil {
		LOG.Exit(err)
	}
}
