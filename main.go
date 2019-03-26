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
	DefaultChpierMethod = "aes-256-cfb"
	DefaultSSHost       = ""
	DefaultSSPort       = 8080
	DefaultCNDNS        = "114.114.114.114"
	DefaultFQDNS        = "8.8.8.8"
)

var configFile = flag.String("config", "", "json cofig file path")
var lHost = flag.String("listen-host", DefaultLHost, "address to listen")
var lPort = flag.Int("listen-port", DefaultLPort, "port to listen")
var ssHost = flag.String("ss-host", DefaultSSHost, "ss server's address")
var ssPort = flag.Int("ss-port", DefaultSSPort, "ss sever's port")
var ssCphierMethod = flag.String("ss-chpier-method", DefaultChpierMethod, "ss server's auth mnethod")
var ssPasswd = flag.String("ss-passwd", "", "ss server's password")
var cnDNS = flag.String("cn-dns", DefaultCNDNS, "dns server in China")
var fqDNS = flag.String("fq-dns", DefaultFQDNS, "dns server not in China")
var verbose = flag.Bool("v", false, "verbose logging")
var clean = flag.Bool("clean", false, "cleanup iptables and ipset")

var LOG *Logger
var ipchain *SNETChain

func main() {
	flag.Parse()

	var logLevel int
	var err error

	if *verbose {
		logLevel = LOG_DEBUG
	} else {
		logLevel = LOG_INFO
	}
	LOG = NewLogger(logLevel)

	ipset, err := NewIPSet()
	exitOnError(err)
	ipchain = NewSNETChain()

	if *clean {
		ipchain.Destroy()
		ipset.Destroy()
		os.Exit(1)
	}

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
	s, err := NewServer(*lHost, *lPort, ssIP, *ssPort, *ssCphierMethod, *ssPasswd)
	exitOnError(err)
	errCh := make(chan error)

	exitOnError(ipset.Init())
	exitOnError(ipset.Bypass(ssIP))

	// order is important
	exitOnError(ipchain.Init())
	exitOnError(ipchain.ByPassIPSet(ipset))
	exitOnError(ipchain.RedirectTCP(*lPort))
	addr := fmt.Sprintf("%s:%d", *lHost, *lPort)
	exitOnError(ipchain.RedirectDNS(addr, *cnDNS))

	dns, err := NewDNS(addr, *cnDNS, *fqDNS)
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
	ipchain.Destroy()
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
