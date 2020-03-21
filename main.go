package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"snet/config"
	"snet/logger"
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
		ctx, cancel := context.WithCancel(context.Background())
		s := NewLocalServer(ctx, c)
		if *clean {
			s.server, err = NewServer(c)
			exitOnError(err, nil)
			s.SetupRedirector()
			s.Clean()
		} else {
			go func() {
				c := make(chan os.Signal, 1)
				signal.Notify(c, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
				l.Info("Got signal:", <-c)
				cancel()
			}()
			go func() {
				c := make(chan os.Signal, 1)
				signal.Notify(c, syscall.SIGHUP)
				for {
					l.Info("Got signal:", <-c)
					cfg, err := config.LoadConfig(*configFile)
					if err != nil {
						l.Error("Failed to reload config:", err)
						continue
					}
					s.cfgChan <- cfg
				}
			}()
			s.Run(nil)
		}
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
