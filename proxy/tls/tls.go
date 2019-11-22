package tls

import (
	_tls "crypto/tls"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"snet/proxy"
)

type Config struct {
	Host    string
	Port    int
	Timeout time.Duration
}

type Server struct {
	Host net.IP
	Port int
	cfg  *Config
}

func (s *Server) Init(c proxy.Config) error {
	s.cfg = c.(*Config)
	ips, err := net.LookupIP(s.cfg.Host)
	if err != nil {
		return err
	}
	s.Host = ips[0]
	s.Port = s.cfg.Port
	return nil
}

func (s *Server) GetProxyIP() net.IP {
	return s.Host
}

func (s *Server) Dial(dstHost string, dstPort int) (net.Conn, error) {
	conn, err := _tls.Dial("tcp", fmt.Sprintf("%s:%d", s.Host, s.Port), &_tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return nil, err
	}
	err = writeDst(conn, dstHost, dstPort)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func writeDst(conn net.Conn, host string, port int) error {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, uint16(len(host)))
	if _, err := conn.Write(buf); err != nil {
		return err
	}
	if _, err := conn.Write([]byte(host)); err != nil {
		return err
	}
	binary.BigEndian.PutUint16(buf, uint16(port))
	if _, err := conn.Write(buf); err != nil {
		return err
	}
	return nil
}

func (s *Server) Pipe(src, dst net.Conn) error {
	defer dst.Close()
	b := make([]byte, 1024)
	for {
		src.SetReadDeadline(time.Now().Add(s.cfg.Timeout))
		n, err := src.Read(b)
		if err != nil {
			return err
		}
		if _, err := dst.Write(b[:n]); err != nil {
			return err
		}
	}
}

func (s *Server) Close() error {
	return nil
}

func init() {
	proxy.Register("tls", new(Server))
}
