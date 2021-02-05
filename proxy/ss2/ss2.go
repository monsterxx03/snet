package ss2

import (
	"encoding/base64"
	"fmt"
	"net"

	"github.com/shadowsocks/go-shadowsocks2/core"
	"github.com/shadowsocks/go-shadowsocks2/socks"

	"snet/proxy"
)

type Config struct {
	Host         net.IP
	Port         int
	CipherMethod string
	Key          string
	Password     string
}

type Server struct {
	Host   net.IP
	Port   int
	cipher core.Cipher
	cfg    *Config
}

func (s *Server) Init(c proxy.Config) error {
	var err error
	s.cfg = c.(*Config)
	s.Host = s.cfg.Host
	s.Port = s.cfg.Port
	var key []byte
	if s.cfg.Key != "" {
		key, err = base64.URLEncoding.DecodeString(s.cfg.Key)
		if err != nil {
			return err
		}
	}
	s.cipher, err = core.PickCipher(s.cfg.CipherMethod, key, s.cfg.Password)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) GetProxyIP() net.IP {
	return s.Host
}

func (s *Server) Dial(dstHost string, dstPort int) (net.Conn, error) {
	ssAddr := fmt.Sprintf("%s:%d", s.Host.String(), s.cfg.Port)
	dst := socks.ParseAddr(fmt.Sprintf("%s:%d", dstHost, dstPort))
	rc, err := net.Dial("tcp", ssAddr)
	if err != nil {
		return nil, err
	}
	rc = s.cipher.StreamConn(rc)
	if _, err := rc.Write(dst); err != nil {
		return nil, fmt.Errorf("failed to send target address: %v", err)
	}
	return rc, nil
}

func (s *Server) Close() error {
	return nil
}

func init() {
	proxy.Register("ss2", new(Server))
}
