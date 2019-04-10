package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"
)

const (
	dnsPort    = 53
	dnsTimeout = 5
	cacheSize  = 5000
)

type DNS struct {
	udpAddr          *net.UDPAddr
	udpListener      *net.UDPConn
	cnDNS            string
	fqDNS            string
	originalResolver []byte
	ipchecker        *IPChecker
	cache            *LRU
}

func NewDNS(laddr, cnDNS, fqDNS string, enableCache bool) (*DNS, error) {
	uaddr, err := net.ResolveUDPAddr("udp", laddr)
	if err != nil {
		return nil, err
	}
	ipchecker, err := NewIPChecker()
	if err != nil {
		return nil, err
	}
	var cache *LRU
	if enableCache {
		cache, err = NewLRU(cacheSize)
		if err != nil {
			return nil, err
		}
	}
	return &DNS{
		udpAddr:   uaddr,
		cnDNS:     cnDNS,
		fqDNS:     fqDNS,
		ipchecker: ipchecker,
		cache:     cache,
	}, nil
}

func (s *DNS) Run() error {
	var err error
	s.udpListener, err = net.ListenUDP("udp", s.udpAddr)
	if err != nil {
		return err
	}
	LOG.Info("listen on udp:", s.udpAddr)
	defer s.udpListener.Close()
	for {
		b := make([]byte, 1024)
		n, uaddr, err := s.udpListener.ReadFromUDP(b)
		if err != nil {
			return err
		}
		go func(uaddr *net.UDPAddr, data []byte) {
			err := s.handle(uaddr, data)
			if err != nil {
				LOG.Err(err)
			}
		}(uaddr, b[:n])
	}
}

func (s *DNS) Shutdown() error {
	if err := s.udpListener.Close(); err != nil {
		return err
	}
	return nil
}

func (s *DNS) handle(reqUaddr *net.UDPAddr, data []byte) error {
	var wg sync.WaitGroup
	var cnData []byte
	var fqData []byte
	dnsQuery, err := s.parse(data)
	if err != nil {
		return err
	}
	if s.cache != nil {
		// only cache A record
		cachedData := s.cache.Get(fmt.Sprintf("%s:%s", dnsQuery.QDomain, dnsQuery.QType))
		if cachedData != nil {
			LOG.Debug("dns cache hit:", dnsQuery.QDomain)
			resp := cachedData.([]byte)
			// rewrite first 2 bytes(dns id)
			resp[0] = data[0]
			resp[1] = data[1]
			if _, err := s.udpListener.WriteToUDP(resp, reqUaddr); err != nil {
				return err
			}
			return nil
		}
	}
	wg.Add(2)
	go func(data []byte) {
		defer wg.Done()
		var err error
		cnData, err = s.queryCN(data)
		if err != nil {
			LOG.Warn("failed to query CN dns:", dnsQuery, err)
		}
	}(data)
	go func(data []byte) {
		defer wg.Done()
		var err error
		fqData, err = s.queryFQ(data)
		if err != nil {
			LOG.Warn("failed to query fq dns:", dnsQuery, err)
		}
	}(data)

	wg.Wait()
	// TODO no need to wait for fq if cn response first and it's a cn ip
	cnMsg, err := s.parse(cnData)
	if err != nil {
		return err
	}
	fqMsg, err := s.parse(fqData)
	if err != nil {
		return err
	}
	LOG.Debug("fq", fqMsg)
	LOG.Debug("cn", cnMsg)
	var raw []byte
	useMsg := cnMsg
	if len(cnMsg.ARecords) >= 1 && s.ipchecker.InChina(cnMsg.ARecords[0].IP) {
		// if cn dns have response and it's an cn ip, we think it's a site in China
		raw = cnData
	} else {
		// use fq dns's response for all ip outside of China
		raw = fqData
		useMsg = fqMsg
	}
	if _, err := s.udpListener.WriteToUDP(raw, reqUaddr); err != nil {
		return err
	}
	if len(useMsg.ARecords) > 0 && s.cache != nil {
		// add to dns cache
		s.cache.Add(fmt.Sprintf("%s:%s", dnsQuery.QDomain, dnsQuery.QType), raw, time.Now().Add(time.Second*time.Duration(useMsg.ARecords[0].TTL)))
	}

	return nil
}

func (s *DNS) parse(data []byte) (*DNSMsg, error) {
	msg, err := NewDNSMsg(data)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (s *DNS) queryCN(data []byte) ([]byte, error) {
	conn, err := net.Dial("udp", fmt.Sprintf("%s:%d", s.cnDNS, dnsPort))
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if err := conn.SetReadDeadline(time.Now().Add(dnsTimeout * time.Second)); err != nil {
		return nil, err
	}
	if _, err = conn.Write(data); err != nil {
		return nil, err
	}
	b := make([]byte, 1024)
	n, err := conn.Read(b)
	if err != nil {
		return nil, err
	}
	return b[0:n], nil
}

func (s *DNS) queryFQ(data []byte) ([]byte, error) {
	// query fq dns by tcp, it will be captured by iptables and go out through ss
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", s.fqDNS, dnsPort))
	if err != nil {
		return nil, err
	}
	if err := conn.SetReadDeadline(time.Now().Add(dnsTimeout * time.Second)); err != nil {
		return nil, err
	}
	defer conn.Close()
	b := make([]byte, 2) // used to hold dns data length
	binary.BigEndian.PutUint16(b, uint16(len(data)))
	if _, err = conn.Write(append(b, data...)); err != nil {
		return nil, err
	}
	b = make([]byte, 2)
	if _, err = conn.Read(b); err != nil {
		return nil, err
	}

	_len := binary.BigEndian.Uint16(b)
	b = make([]byte, _len)
	if _, err = conn.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}
