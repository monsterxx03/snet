package main

import (
	"log"
	"net"
	"syscall"
)

type Server struct {
	listener *net.TCPListener
}

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
	err := getDst(conn)
	if err != nil {
		return err
	}
	b := make([]byte, 1024)
	n, err := conn.Read(b)
	if err != nil {
		return err
	}
	log.Printf("%x\n", b[:n])
	return nil
}

func (s *Server) Shutdown() error {
	return s.listener.Close()
}

func getDst(conn *net.TCPConn) error {
	f, err := conn.File()
	if err != nil {
		return nil
	}
	// f is a copy of tcp connection's underlying fd, close it won't affect current connection
	defer f.Close()
	fd := f.Fd() // returned fd is in blocking mode
	if err := syscall.SetNonblock(int(fd), true); err != nil {
		return err
	}
	return nil
}
