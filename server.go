package main

import (
	"log"
	"net"
	"syscall"
	"fmt"
	ss "github.com/shadowsocks/shadowsocks-go/shadowsocks"
)

type UPStreamServer interface{
}

type SSServer struct {
	Host string
	Port int
	CipherMethod string
	Pasword string
}

type Server struct {
	listener *net.TCPListener
	upstream *UPStreamServer
}

const (
	SO_ORIGINAL_DST = 80 // /usr/includ/linux/netfilter_ipv4.h
)

func NewServer(addr string) (*Server, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	log.Println("Listen on ", addr)
	return &Server{listener: ln.(*net.TCPListener)}, nil
}

func (s *Server) Run() error {
	for {
		conn, err := s.listener.AcceptTCP()
		if err != nil {
			return err
		}
		go func(conn *net.TCPConn) {
			defer conn.Close()
			if err := s.HandleConn(conn); err != nil {
				log.Println(err)
			}
		}(conn)
	}
}

func (s *Server) HandleConn(conn *net.TCPConn) error {
	log.Println("Connection from ", conn.RemoteAddr())
	dst, err := getDst(conn)
	if err != nil {
		return err
	}
	log.Println(dst)
	cipher, err := ss.NewCipher("aes-256-cfb", "passwd")
	if err != nil {
		return err
	}
	remoteConn, err := ss.Dial(dst, "server", cipher)  // connect to ss server
	if err != nil {
		return err
	}
	go ss.PipeThenClose(conn, remoteConn, nil)
	ss.PipeThenClose(remoteConn, conn, nil)
	log.Println("close connection to", dst)
	return nil
}

func (s *Server) Shutdown() error {
	return s.listener.Close()
}

func getDst(conn *net.TCPConn) (dst string, err error) {
	f, err := conn.File()
	if err != nil {
		return "", err
	}
	// f is a copy of tcp connection's underlying fd, close it won't affect current connection
	defer f.Close()
	fd := f.Fd() // returned fd is in blocking mode
	if err := syscall.SetNonblock(int(fd), true); err != nil {
		return "", err
	}
	addr, err := syscall.GetsockoptIPv6Mreq(int(fd), syscall.IPPROTO_IP, SO_ORIGINAL_DST)
	if err != nil  {
		return "", err
	}
	// ipv4 addr is bytes 5 to 8
	// port number is bytes 3 and 3
	host := fmt.Sprintf("%d.%d.%d.%d:%d",
		addr.Multiaddr[4],
		addr.Multiaddr[5],
		addr.Multiaddr[6],
		addr.Multiaddr[7],
		uint16(addr.Multiaddr[2])<<8+uint16(addr.Multiaddr[3]))
	return host, err
}
