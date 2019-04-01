package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	ss "github.com/shadowsocks/shadowsocks-go/shadowsocks"
	"net"
	"syscall"
)

const (
	SO_ORIGINAL_DST = 80 // /usr/includ/linux/netfilter_ipv4.h
)

type SSServer struct {
	Host   string
	Port   int
	cipher *ss.Cipher
}

type TCPServer struct {
	lHost    string
	lPort    int
	listener *net.TCPListener
	ss       *SSServer
}

func (s *TCPServer) Dial(tgtAddr string) (*ss.Conn, error) {
	return ss.Dial(tgtAddr, fmt.Sprintf("%s:%d", s.ss.Host, s.ss.Port), s.ss.cipher.Copy())
}

func (s *TCPServer) Pipe(src, dst net.Conn) {
	ss.PipeThenClose(src, dst, nil)
}

func NewTCPServer(lHost string, lPort int, ssHost string, ssPort int, ssCphierMethod string, ssPasswd string) (*TCPServer, error) {
	addr := fmt.Sprintf("%s:%d", lHost, lPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	LOG.Info("Listen on tcp:", addr)

	cipher, err := ss.NewCipher(ssCphierMethod, ssPasswd)
	if err != nil {
		return nil, err
	}
	ssServer := &SSServer{
		Host:   ssHost,
		Port:   ssPort,
		cipher: cipher,
	}
	return &TCPServer{
		lHost:    lHost,
		lPort:    lPort,
		listener: ln.(*net.TCPListener),
		ss:       ssServer,
	}, nil
}

func (s *TCPServer) Run() error {
	for {
		conn, err := s.listener.AcceptTCP()
		if err != nil {
			return err
		}
		go func(conn *net.TCPConn) {
			defer conn.Close()
			if err := s.HandleConn(conn); err != nil {
				LOG.Err(err)
			}
		}(conn)
	}
}

func (s *TCPServer) HandleConn(conn *net.TCPConn) error {
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

func (s *TCPServer) Shutdown() error {
	return s.listener.Close()
}

func getDst(conn *net.TCPConn) (dstHost string, dstPort int, err error) {
	f, err := conn.File()
	if err != nil {
		return "", -1, err
	}
	// f is a copy of tcp connection's underlying fd, close it won't affect current connection
	defer f.Close()
	fd := f.Fd() // from go1.11 returned fd is no longer in blocking mode
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

type UDPServer struct {
	lHost string
	lPort int
}

func NewUDPServer(lHost string, lPort int) (*UDPServer, error) {
	return &UDPServer{
		lHost: lHost,
		lPort: lPort,
	}, nil
}

func (s *UDPServer) Run() error {
	addr := fmt.Sprintf("%s:%d", s.lHost, s.lPort)
	LOG.Info("listen on udp:", addr)
	// ListenConfig added in go 1.11
	lc := net.ListenConfig{
		Control: func(network string, addr string, c syscall.RawConn) error {
			var opErr error
			if err := c.Control(func(fd uintptr) {
				opErr = syscall.SetsockoptInt(int(fd), syscall.SOL_IP, syscall.IP_TRANSPARENT, 1)
			}); err != nil {
				return err
			}
			if opErr != nil {
				return opErr
			}
			if err := c.Control(func(fd uintptr) {
				opErr = syscall.SetsockoptInt(int(fd), syscall.SOL_IP, syscall.IP_RECVORIGDSTADDR, 1)
			}); err != nil {
				return err
			}
			return opErr
		}}
	lp, err := lc.ListenPacket(context.Background(), "udp", addr)
	if err != nil {
		return err
	}
	conn := lp.(*net.UDPConn)
	for {
		// From https://github.com/LiamHaworth/go-tproxy/blob/master/tproxy_udp.go
		b := make([]byte, 1024)
		oob := make([]byte, 1024)
		n, oobn, _, _, err := conn.ReadMsgUDP(b, oob)
		if err != nil {
			return err
		}
		msgs, err := syscall.ParseSocketControlMessage(oob[:oobn])
		if err != nil {
			return err
		}
		var orgDstAddr *net.UDPAddr
		for _, msg := range msgs {
			if msg.Header.Level == syscall.SOL_IP && msg.Header.Type == syscall.IP_RECVORIGDSTADDR {
				orgDstAddrRaw := &syscall.RawSockaddrInet4{}
				if err = binary.Read(bytes.NewReader(msg.Data), binary.LittleEndian, orgDstAddrRaw); err != nil {
					return err
				}
				addrBytes := orgDstAddrRaw.Addr
				port := orgDstAddrRaw.Port
				orgDstAddr = &net.UDPAddr{
					IP:   net.IPv4(addrBytes[0], addrBytes[1], addrBytes[2], addrBytes[3]),
					Port: int((port&0xff)<<8 + port>>8), // swap lower byte and high byte
				}
			}
		}
		if orgDstAddr == nil {
			return fmt.Errorf("Failed to get original udp dst addr")
		}
		fmt.Println("original udp dst addr:", orgDstAddr)
		fmt.Println("receved data:", b[:n])
	}
}

func (s *UDPServer) Shutdown() error {
	return nil
}
