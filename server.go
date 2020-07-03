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
	"snet/sniffer"
	"snet/stats"
	"snet/utils"
)

const (
	SO_ORIGINAL_DST = 80 // /usr/includ/linux/netfilter_ipv4.h
)

type HostBytesMap struct {
	sync.RWMutex
	m map[string]uint64
}

type Server struct {
	ctx      context.Context
	cfg      *config.Config
	listener *net.TCPListener
	proxy    proxy.Proxy
	timeout  time.Duration

	// Total number from start
	HostRxBytesTotal *HostBytesMap
	HostTxBytesTotal *HostBytesMap
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
	if c.EnableStats {
		rxCh = make(chan *stats.P, 1)
		txCh = make(chan *stats.P, 1)
	}
	return &Server{
		ctx:              ctx,
		cfg:              c,
		listener:         ln.(*net.TCPListener),
		proxy:            p,
		timeout:          time.Duration(c.ProxyTimeout) * time.Second,
		HostRxBytesTotal: &HostBytesMap{m: make(map[string]uint64)},
		HostTxBytesTotal: &HostBytesMap{m: make(map[string]uint64)},
		rxCh:             rxCh,
		txCh:             txCh,
	}, nil
}

func (s *Server) Run() error {
	l.Infof("Proxy server listen on tcp %s:%d", s.cfg.LHost, s.cfg.LPort)
	if s.cfg.EnableStats {
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
			s.HostRxBytesTotal.Lock()
			if _, ok := s.HostRxBytesTotal.m[p.Host]; ok {
				s.HostRxBytesTotal.m[p.Host] += p.Rx
			} else {
				s.HostRxBytesTotal.m[p.Host] = p.Rx
			}
			s.HostRxBytesTotal.Unlock()
		case p := <-s.txCh:
			s.HostTxBytesTotal.Lock()
			if _, ok := s.HostTxBytesTotal.m[p.Host]; ok {
				s.HostTxBytesTotal.m[p.Host] += p.Tx
			} else {
				s.HostTxBytesTotal.m[p.Host] = p.Tx
			}
			s.HostTxBytesTotal.Unlock()
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
	defer remoteConn.Close()
	var sn *sniffer.Sniffer
	if s.cfg.EnableStats {
		sn = sniffer.NewSniffer(s.cfg.StatsEnableTLSSNISniffer, s.cfg.StatsEnableHTTPHostSniffer)
	}
	if err := utils.Pipe(s.ctx, conn, remoteConn, s.timeout, s.rxCh, s.txCh, dstHost, dstPort, sn); err != nil {
		l.Error(err)
	}
	return nil
}

func (s *Server) Shutdown() error {
	err := s.listener.Close()
	if err != nil {
		return err
	}
	l.Info("redirector tcp server shutdown")
	return nil
}
