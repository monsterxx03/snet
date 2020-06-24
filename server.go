package main

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"snet/config"
	"snet/proxy"
	"snet/redirector"
	"snet/stats"
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
	HostRxBytesTotal map[string]uint64
	HostTxBytesTotal map[string]uint64
	rxLock           sync.Mutex
	txLock           sync.Mutex
	rxCh             chan *stats.P
	txCh             chan *stats.P
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
	var rxCh, txCh chan *stats.P
	if c.EnableStat {
		rxCh = make(chan *stats.P, 1)
		txCh = make(chan *stats.P, 1)
	}
	return &Server{
		ctx:              ctx,
		cfg:              c,
		listener:         ln.(*net.TCPListener),
		proxy:            p,
		timeout:          time.Duration(c.ProxyTimeout) * time.Second,
		HostRxBytesTotal: make(map[string]uint64),
		HostTxBytesTotal: make(map[string]uint64),
		rxCh:             rxCh,
		txCh:             txCh,
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
		case p := <-s.rxCh:
			s.rxLock.Lock()
			if _, ok := s.HostRxBytesTotal[p.Host]; ok {
				s.HostRxBytesTotal[p.Host] += p.Rx
			} else {
				s.HostRxBytesTotal[p.Host] = p.Rx
			}
			s.rxLock.Unlock()
		case p := <-s.txCh:
			s.txLock.Lock()
			if _, ok := s.HostTxBytesTotal[p.Host]; ok {
				s.HostTxBytesTotal[p.Host] += p.Tx
			} else {
				s.HostTxBytesTotal[p.Host] = p.Tx
			}
			s.txLock.Unlock()
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
	// TODO do intercept to figure out tls/http server_name
	if err := utils.Pipe(s.ctx, conn, remoteConn, s.timeout, s.rxCh, s.txCh, fmt.Sprintf("%s:%d", dstHost, dstPort)); err != nil {
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
