package main

import (
	"fmt"
	"net"
	"time"

	"snet/proxy"
	"snet/redirector"
	"snet/utils"
)

const (
	SO_ORIGINAL_DST = 80 // /usr/includ/linux/netfilter_ipv4.h
)

type Server struct {
	listener *net.TCPListener
	proxy    proxy.Proxy
	timeout  time.Duration
}

func NewServer(c *Config) (*Server, error) {
	addr := fmt.Sprintf("%s:%d", c.LHost, c.LPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	l.Info("Proxy server listen on tcp:", addr)

	if err != nil {
		return nil, err
	}
	p, err := proxy.Get(c.ProxyType)
	if err != nil {
		return nil, err
	}
	if err := p.Init(genConfigByType(c, c.ProxyType)); err != nil {
		return nil, err
	}
	return &Server{
		listener: ln.(*net.TCPListener),
		proxy:    p,
		timeout:  time.Duration(c.ProxyTimeout) * time.Second,
	}, nil
}

func (s *Server) Run() error {
	for {
		conn, err := s.listener.AcceptTCP()
		if err != nil {
			return err
		}
		go func(conn *net.TCPConn) {
			if err := s.handle(conn); err != nil {
				l.Warn(err)
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
	if err := remoteConn.SetDeadline(time.Now().Add(s.timeout)); err != nil {
		return err
	}
	defer remoteConn.Close()
	if err := utils.Pipe(conn, remoteConn); err != nil {
		return err
	}
	return nil
}

func (s *Server) Shutdown() error {
	return s.listener.Close()
}
