package main

import (
	"context"
	"fmt"
	"net"
	"time"

	"snet/config"
	"snet/proxy"
	"snet/redirector"
	"snet/utils"
)

const (
	SO_ORIGINAL_DST = 80 // /usr/includ/linux/netfilter_ipv4.h
)

type Server struct {
	ctx      context.Context
	cfg      *config.Config
	listener *net.TCPListener
	proxy    proxy.Proxy
	timeout  time.Duration
}

func NewServer(ctx context.Context, c *config.Config) (*Server, error) {
	addr := fmt.Sprintf("%s:%d", c.LHost, c.LPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}
	p, err := proxy.Get(c.ProxyType)
	if err != nil {
		return nil, err
	}
	cfg, err := genConfigByType(c, c.ProxyType)
	if err != nil {
		return nil, err
	}
	if err := p.Init(cfg); err != nil {
		return nil, err
	}
	return &Server{
		ctx:      ctx,
		cfg:      c,
		listener: ln.(*net.TCPListener),
		proxy:    p,
		timeout:  time.Duration(c.ProxyTimeout) * time.Second,
	}, nil
}

func (s *Server) Run() error {
	l.Infof("Proxy server listen on tcp %s:%d", s.cfg.LHost, s.cfg.LPort)
	for {
		conn, err := s.listener.AcceptTCP()
		if err != nil {
			return err
		}
		go func(conn *net.TCPConn) {
			if err := s.handle(conn); err != nil {
				l.Error(err)
			}
		}(conn)
	}
}

func (s *Server) handle(conn *net.TCPConn) error {
	defer conn.Close()
	dstHost, dstPort, err := redirector.GetDstAddr(conn)
	if err != nil {
		return err
	}
	remoteConn, err := s.proxy.Dial(dstHost, dstPort)
	if err != nil {
		return err
	}
	// if intercept is enabled, use i to replace conn
	// i := proxy.NewIntercept(conn, dstHost, dstPort, l)
	defer remoteConn.Close()
	if err := utils.Pipe(s.ctx, conn, remoteConn, s.timeout); err != nil {
		l.Error(err)
	}
	return nil
}

func (s *Server) Shutdown() error {
	err := s.listener.Close()
	if err != nil {
		return err
	}
	l.Info("tcp server shutdown")
	return nil
}
