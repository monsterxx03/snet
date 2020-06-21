package main

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"snet/config"
	"snet/proxy"
	"snet/redirector"
	"snet/utils"
)

const (
	SO_ORIGINAL_DST = 80 // /usr/includ/linux/netfilter_ipv4.h
)

type Server struct {
	ctx      context.Context
	cfg      *config.Config
	listener *net.TCPListener
	proxy    proxy.Proxy
	timeout  time.Duration

	// Total number from start
	RxBytesTotal uint64
	TxBytesTotal uint64
	rxBytesCh    chan uint64
	txBytesCh    chan uint64
}

func NewServer(ctx context.Context, c *config.Config) (*Server, error) {
	addr := fmt.Sprintf("%s:%d", c.LHost, c.LPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}
	p, err := proxy.Get(c.ProxyType)
	if err != nil {
		return nil, err
	}
	cfg, err := genConfigByType(c, c.ProxyType)
	if err != nil {
		return nil, err
	}
	if err := p.Init(cfg); err != nil {
		return nil, err
	}
	return &Server{
		ctx:       ctx,
		cfg:       c,
		listener:  ln.(*net.TCPListener),
		proxy:     p,
		timeout:   time.Duration(c.ProxyTimeout) * time.Second,
		rxBytesCh: make(chan uint64),
		txBytesCh: make(chan uint64),
	}, nil
}

func (s *Server) Run() error {
	l.Infof("Proxy server listen on tcp %s:%d", s.cfg.LHost, s.cfg.LPort)
	if s.cfg.EnableStat {
		go s.receiveStat()
	}
	for {
		conn, err := s.listener.AcceptTCP()
		if err != nil {
			return err
		}
		go func(conn *net.TCPConn) {
			if err := s.handle(conn); err != nil {
				l.Error(err)
			}
		}(conn)
	}
}

func (s *Server) receiveStat() {
	for {
		select {
		case n := <-s.rxBytesCh:
			atomic.AddUint64(&s.RxBytesTotal, uint64(n))
		case n := <-s.txBytesCh:
			atomic.AddUint64(&s.TxBytesTotal, uint64(n))
		case <-s.ctx.Done():
			return
		}
	}

}

func (s *Server) handle(conn *net.TCPConn) error {
	defer conn.Close()
	dstHost, dstPort, err := redirector.GetDstAddr(conn)
	if err != nil {
		return err
	}
	remoteConn, err := s.proxy.Dial(dstHost, dstPort)
	if err != nil {
		return err
	}
	// if intercept is enabled, use i to replace conn
	// i := proxy.NewIntercept(conn, dstHost, dstPort, l)
	defer remoteConn.Close()
	err = utils.Pipe(conn, remoteConn, s.timeout, s.cfg.EnableStat, s.rxBytesCh, s.txBytesCh)
	if err != nil {
		l.Error(err)
	}
	return nil
}

func (s *Server) Shutdown() error {
	err := s.listener.Close()
	if err != nil {
		return err
	}
	l.Info("tcp server shutdown")
	return nil
}
