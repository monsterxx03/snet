package main

import (
	"flag"
	"net"
	"os"
	"os/signal"
	"syscall"
)

//go:generate go run chnroutes_generate.go

var lHost = flag.String("listen-host", "127.0.0.1", "address to listen")
var lPort = flag.Int("listen-port", 1111, "port to listen")
var ssHost = flag.String("ss-host", "127.0.0.1", "ss server's address")
var ssPort = flag.Int("ss-port", 8080, "ss sever's port")
var ssCphierMethod = flag.String("ss-chpier-method", "aes-256-cfb", "ss server's auth mnethod")
var ssPasswd = flag.String("ss-passwd", "", "ss server's password")
var cnDNS = flag.String("cn-dns", "114.114.114.114", "dns server in China")
var fqDNS = flag.String("fq-dns", "8.8.8.8", "dns server not in China")

var LOG *Logger

func main() {
	flag.Parse()
	LOG = NewLogger(LOG_ERR)
	if *ssPasswd == "" {
		LOG.Exit("ss-passwd is required")
	}
	ips, err := net.LookupIP(*ssHost)
	exitOnError(err)
	ssIP := ips[0].String()
	s, err := NewServer(*lHost, *lPort, ssIP, *ssPort, *ssCphierMethod, *ssPasswd)
	exitOnError(err)
	errCh := make(chan error)

	ipset, err := NewIPSet()
	exitOnError(err)
	exitOnError(ipset.Init())
	exitOnError(ipset.Bypass(ssIP))
	exitOnError(setBypassRule(ipset.Name))
	exitOnError(setRedirectRule(*lPort))

	dns, err := NewDNS(*cnDNS, *fqDNS)
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
		if err := delByassRule(ipset.Name); err != nil {
			errCh <- err
			return
		}
		if err := delRedirectRule(*lPort); err != nil {
			errCh <- err
			return
		}
		errCh <- ipset.Destroy()
	}()

	if err := <-errCh; err != nil {
		LOG.Err(err)
	}
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
