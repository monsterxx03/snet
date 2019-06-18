package main

import (
	"fmt"
	"net"
	"snet/proxy"
	"syscall"
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
	dstHost, dstPort, err := getDst(conn)
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

func getDst(conn *net.TCPConn) (dstHost string, dstPort int, err error) {
	f, err := conn.File()
	if err != nil {
		return "", -1, err
	}
	// f is a copy of tcp connection's underlying fd, close it won't affect current connection
	defer f.Close()
	fd := f.Fd()
	addr, err := syscall.GetsockoptIPv6Mreq(int(fd), syscall.IPPROTO_IP, SO_ORIGINAL_DST)
	if err != nil {
		return "", -1, err
	}
	// ipv4 addr is bytes 5 to 8
	// port number is bytes 3 and 3
	host := fmt.Sprintf("%d.%d.%d.%d",
		addr.Multiaddr[4],
		addr.Multiaddr[5],
		addr.Multiaddr[6],
		addr.Multiaddr[7],
	)
	return host, int(addr.Multiaddr[2])<<8 + int(addr.Multiaddr[3]), err
}
