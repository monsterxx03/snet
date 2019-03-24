package main

import (
	"fmt"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"net"
)

var lHost = flag.String("listen-host", "127.0.0.1", "address to listen")
var lPort = flag.Int("listen-port", 1111, "port to listen")
var ssHost = flag.String("ss-host", "127.0.0.1", "ss server's address")
var ssPort = flag.Int("ss-port", 8080, "ss sever's port")
var ssCphierMethod = flag.String("ss-chpier-method", "aes-256-cfb", "ss server's auth mnethod")
var ssPasswd = flag.String("ss-passwd", "", "ss server's password")
var cnDNS = flag.String("cn-dns", "114.114.114.114", "dns server in China")
var fqDNS = flag.String("fq-dns", "8.8.8.8", "dns server not in China")

func main() {
	flag.Parse()
	if *ssPasswd == "" {
		log.Println("ss-passwd is required")
		os.Exit(1)
	}
	ips, err := net.LookupIP(*ssHost)
	exitOnError(err)
	ssIP := ips[0].String()
	s, err := NewServer(*lHost, *lPort, ssIP, *ssPort, *ssCphierMethod, *ssPasswd)
	if err != nil {
		log.Println(err)
	}
	errCh := make(chan error)

	ipset, err := NewIPSet()
	exitOnError(err)
	exitOnError(ipset.Init())
	exitOnError(ipset.Bypass(ssIP))
	exitOnError(setBypassRule(ipset.Name))
	exitOnError(setRedirectRule(*lPort))
	addr := fmt.Sprintf("%s:%d", *lHost, *lPort)
	exitOnError(setDNSRule(addr))

	dns, err := NewDNS(*lPort, *fqDNS)
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
		log.Println("Got signal: ", <-c)
		if err := delDNSRule(addr); err != nil {
			errCh <- err
			return
		}
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
		log.Println(err)
	}
	if s.Shutdown() != nil {
		log.Println("Error during shutdown", err)
	}
}


func exitOnError(err error) {
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
