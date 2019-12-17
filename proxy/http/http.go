package http

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net"

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
	auth string
	cfg  *Config
}

var OK_MSG = "HTTP/1.1 200"

func (s *Server) Init(c proxy.Config) error {
	s.cfg = c.(*Config)
	s.Host = s.cfg.Host
	s.Port = s.cfg.Port
	s.auth = base64.StdEncoding.EncodeToString([]byte(s.cfg.AuthUser + ":" + s.cfg.AuthPassword))
	return nil
}

func (s *Server) GetProxyIP() net.IP {
	return s.Host
}

func (s *Server) Dial(dstHost string, dstPort int) (net.Conn, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", s.Host, s.Port))
	if err != nil {
		return nil, err
	}
	handshake := fmt.Sprintf("Connect %s:%d HTTP/1.1\r\nProxy-Authorization: Basic %s\r\n\r\n",
		dstHost, dstPort, s.auth)
	_, err = conn.Write([]byte(handshake))
	if err != nil {
		return nil, err
	}
	b := make([]byte, 1024)
	n, err := conn.Read(b)
	if err != nil {
		return nil, err
	}
	resp := string(b[:n])
	if len(resp) < len(OK_MSG) || resp[:len(OK_MSG)] != OK_MSG {
		return nil, errors.New("http tunnel handshake failed:" + resp)
	}
	return conn, nil
}

func (s *Server) Close() error {
	return nil
}

func init() {
	proxy.Register("http", new(Server))
}
