package socks5

import (
	"fmt"
	"net"

	xproxy "golang.org/x/net/proxy"

	"snet/proxy"
)

type Config struct {
	Host         net.IP
	Port         int
	AuthUser     string
	AuthPassword string
}

type Server struct {
	Host net.IP
	Port int
	dial xproxy.Dialer
	cfg  *Config
}

func (s *Server) Init(c proxy.Config) error {
	s.cfg = c.(*Config)
	s.Host = s.cfg.Host
	s.Port = s.cfg.Port

	auth := new(xproxy.Auth)
	if s.cfg.AuthUser != "" {
		auth.User = s.cfg.AuthUser
		auth.Password = s.cfg.AuthPassword
	}
	dial, err := xproxy.SOCKS5("tcp", fmt.Sprintf("%s:%d", s.Host, s.Port), auth, xproxy.Direct)
	if err != nil {
		return err
	}
	s.dial = dial
	return nil
}

func (s *Server) GetProxyIP() net.IP {
	return s.Host
}

func (s *Server) Dial(dstHost string, dstPort int) (net.Conn, error) {
	conn, err := s.dial.Dial("tcp", fmt.Sprintf("%s:%d", dstHost, dstPort))
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (s *Server) Close() error {
	return nil
}

func init() {
	proxy.Register("socks5", new(Server))
}
