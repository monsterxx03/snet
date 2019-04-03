package ss

import (
	"fmt"
	ss "github.com/shadowsocks/shadowsocks-go/shadowsocks"
	"net"
	proxy "snet/proxy"
)

type Config struct {
	Host         string
	Port         int
	CipherMethod string
	Password     string
}

type Server struct {
	Host   net.IP
	Port   int
	cipher *ss.Cipher
	cfg    *Config
}

func (s *Server) Init(c proxy.Config) error {
	s.cfg = c.(*Config)
	ips, err := net.LookupIP(s.cfg.Host)
	if err != nil {
		return err
	}
	s.Host = ips[0]
	s.Port = s.cfg.Port
	s.cipher, err = ss.NewCipher(s.cfg.CipherMethod, s.cfg.Password)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) GetProxyIP() net.IP {
	return s.Host
}

func (s *Server) Dial(dstHost string, dstPort int) (net.Conn, error) {
	dst := fmt.Sprintf("%s:%d", dstHost, dstPort)
	ssAddr := fmt.Sprintf("%s:%d", s.Host.String(), s.cfg.Port)
	return ss.Dial(dst, ssAddr, s.cipher.Copy())
}

func (s *Server) Pipe(src, dst net.Conn) error {
	ss.PipeThenClose(src, dst, nil)
	return nil
}

func (s *Server) Close() error {
	return nil
}

func init() {
	proxy.Register("ss", new(Server))
}
