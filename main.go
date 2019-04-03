package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
)

//go:generate go run chnroutes_generate.go

const (
	DefaultLHost        = "127.0.0.1"
	DefaultLPort        = 1111
	DefaultProxyType    = "ss"
	DefaultChpierMethod = "aes-256-cfb"
	DefaultSSHost       = ""
	DefaultSSPort       = 8080
	DefaultCNDNS        = "223.6.6.6"
	DefaultFQDNS        = "8.8.8.8"
	DefaultMode         = "local"
)

var (
	sha1Ver string
	buildAt string
)

var configFile = flag.String("config", "", "json cofig file path")
var lHost = flag.String("listen-host", DefaultLHost, "address to listen")
var lPort = flag.Int("listen-port", DefaultLPort, "port to listen")
var ssHost = flag.String("ss-host", DefaultSSHost, "ss server's address")
var ssPort = flag.Int("ss-port", DefaultSSPort, "ss sever's port")
var ssCphierMethod = flag.String("ss-chpier-method", DefaultChpierMethod, "ss server's auth mnethod")
var ssPasswd = flag.String("ss-passwd", "", "ss server's password")
var cnDNS = flag.String("cn-dns", DefaultCNDNS, "dns server in China")
var fqDNS = flag.String("fq-dns", DefaultFQDNS, "dns server out of China")
var enableDNSCache = flag.Bool("enable-dns-cache", true, "cache dns query result based on ttl")
var mode = flag.String("mode", DefaultMode, "local or router")
var verbose = flag.Bool("v", false, "verbose logging")
var version = flag.Bool("version", false, "print version only")
var clean = flag.Bool("clean", false, "cleanup iptables and ipset")

var LOG *Logger
var ProxyType string = DefaultProxyType

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

	config := &Config{}
	if *configFile != "" {
		config, err = LoadConfig(*configFile)
		exitOnError(err)
		lHost = &config.LHost
		lPort = &config.LPort
		ssHost = &config.SSHost
		ssPort = &config.SSPort
		ssCphierMethod = &config.SSCphierMethod
		ssPasswd = &config.SSPasswd
		cnDNS = &config.CNDNS
		fqDNS = &config.FQDNS
		enableDNSCache = &config.EnableDNSCache
		if config.ProxyType != "" {
			ProxyType = config.ProxyType
		}
		if config.Mode != "" {
			mode = &config.Mode
		}
	}

	dnsPort := *lPort + 100

	if *clean {
		cleanIptableRules(*mode, *lHost, *lPort, dnsPort, setName)
		ipset.Destroy()
		os.Exit(0)
	}

	if *ssPasswd == "" {
		LOG.Exit("ss-passwd is required")
	}
	if *ssHost == "" {
		LOG.Exit("ss-host is required")
	}
	ips, err := net.LookupIP(*ssHost)
	exitOnError(err)
	ssIP := ips[0].String()
	s, err := NewServer(config)
	exitOnError(err)
	errCh := make(chan error)

	// order is important
	exitOnError(ipset.Init())
	exitOnError(ipset.Bypass(ssIP))
	setupIptableRules(*mode, *lHost, *lPort, dnsPort, *cnDNS, setName)

	addr := fmt.Sprintf("%s:%d", *lHost, dnsPort)
	dns, err := NewDNS(addr, *cnDNS, *fqDNS, *enableDNSCache)
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
	cleanIptableRules(*mode, *lHost, *lPort, dnsPort, setName)
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
