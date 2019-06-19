package main

import (
	"fmt"
	"net"

	"snet/proxy"
	"snet/redirector"
)

const (
	SO_ORIGINAL_DST = 80 // /usr/includ/linux/netfilter_ipv4.h
)

type Server struct {
	listener *net.TCPListener
	proxy    proxy.Proxy
}

func NewServer(c *Config) (*Server, error) {
	addr := fmt.Sprintf("%s:%d", c.LHost, c.LPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	LOG.Info("Listen on tcp:", addr)

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
	}, nil
}

func (s *Server) Run() error {
	for {
		conn, err := s.listener.AcceptTCP()
		if err != nil {
			return err
		}
		go func(conn *net.TCPConn) {
			defer conn.Close()
			if err := s.handle(conn); err != nil {
				LOG.Err(err)
			}
		}(conn)
	}
}

func (s *Server) handle(conn *net.TCPConn) error {
	dstHost, dstPort, err := redirector.GetDstAddr(conn)
	fmt.Println(dstHost, dstPort)
	if err != nil {
		return err
	}
	remoteConn, err := s.proxy.Dial(dstHost, dstPort)
	if err != nil {
		return err
	}
	go s.proxy.Pipe(conn, remoteConn)
	s.proxy.Pipe(remoteConn, conn)
	return nil
}

func (s *Server) Shutdown() error {
	return s.listener.Close()
}
