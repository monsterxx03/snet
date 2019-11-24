package main

import (
	"crypto/tls"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"net"
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

var tlsserver = flag.String("tlsserver", "", "run as tls server, eg: 0.0.0.0:9999")
var tlskey = flag.String("tlskey", "server.key", "private key used in tls")
var tlscrt = flag.String("tlscrt", "server.pem", "cert used in tls")
var tlstoken = flag.String("tlstoken", "", "used in tlsserver mode")

var configFile = flag.String("config", "", "json config file path, only used when working as client")
var clean = flag.Bool("clean", false, "cleanup iptables and ipset")
var version = flag.Bool("version", false, "print version only")
var verbose = flag.Bool("v", false, "verbose output")
var l *logger.Logger

func runClient() {
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

	exitOnError(redir.SetupRules(config.Mode, config.LHost, config.LPort, dnsPort, config.CNDNS), cleanupCallback)

	addr := fmt.Sprintf("%s:%d", config.LHost, dnsPort)
	dns, err := dns.NewServer(addr, config.CNDNS, config.FQDNS, config.EnableDNSCache, config.EnforceTTL, config.DisableQTypes, config.ForceFQ, config.HostMap, config.BlockHostFile, config.BlockHosts, Chnroutes, l)
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

func runTLSServer() {
	if *tlstoken == "" {
		exitOnError(errors.New("missing -tlstoken"), nil)
	}
	cert, err := tls.LoadX509KeyPair(*tlscrt, *tlskey)
	exitOnError(err, nil)
	config := &tls.Config{Certificates: []tls.Certificate{cert}}
	ln, err := tls.Listen("tcp", *tlsserver, config)
	exitOnError(err, nil)
	l.Info("TLS server running:", *tlsserver)
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
			if string(b) != *tlstoken {
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
			go pipe(conn, dstConn)
			pipe(dstConn, conn)
		}(conn)
	}
}

func pipe(src, dst net.Conn) error {
	defer dst.Close()
	b := make([]byte, 1024)
	for {
		n, err := src.Read(b)
		if err != nil {
			return err
		}
		if _, err := dst.Write(b[:n]); err != nil {
			return err
		}
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
	if *tlsserver != "" {
		runTLSServer()
	} else {
		runClient()
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
