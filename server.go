package main

import (
	"fmt"
	ss "github.com/shadowsocks/shadowsocks-go/shadowsocks"
	"log"
	"net"
	"syscall"
)

const (
	SO_ORIGINAL_DST = 80 // /usr/includ/linux/netfilter_ipv4.h
)

type Server struct {
	lHost    string
	lPort    int
	listener *net.TCPListener
	ss       *SSServer
}

type SSServer struct {
	Host   string
	Port   int
	cipher *ss.Cipher
}

func (s *Server) Dial(tgtAddr string) (*ss.Conn, error) {
	return ss.Dial(tgtAddr, fmt.Sprintf("%s:%d", s.ss.Host, s.ss.Port), s.ss.cipher.Copy())
}

func (s *Server) Pipe(src, dst net.Conn) {
	ss.PipeThenClose(src, dst, nil)
}

func NewServer(lHost string, lPort int, ssHost string, ssPort int, ssCphierMethod string, ssPasswd string) (*Server, error) {
	addr := fmt.Sprintf("%s:%d", lHost, lPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	log.Println("Listen on ", addr)

	cipher, err := ss.NewCipher(ssCphierMethod, ssPasswd)
	if err != nil {
		return nil, err
	}
	ssServer := &SSServer{
		Host:   ssHost,
		Port:   ssPort,
		cipher: cipher,
	}
	return &Server{
		lHost:    lHost,
		lPort:    lPort,
		listener: ln.(*net.TCPListener),
		ss:       ssServer,
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
			if err := s.HandleConn(conn); err != nil {
				log.Println(err)
			}
		}(conn)
	}
}

func (s *Server) HandleConn(conn *net.TCPConn) error {
	dstHost, dstPort, err := getDst(conn)
	if err != nil {
		return err
	}
	remoteConn, err := s.Dial(fmt.Sprintf("%s:%d", dstHost, dstPort))
	if err != nil {
		return err
	}
	go s.Pipe(conn, remoteConn)
	s.Pipe(remoteConn, conn)
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
	fd := f.Fd() // returned fd is in blocking mode
	if err := syscall.SetNonblock(int(fd), true); err != nil {
		return "", -1, err
	}
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
