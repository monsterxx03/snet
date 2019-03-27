package main

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"net"
	"sync"
	"time"
)

const (
	dnsPort      = 53
	resolverFile = "/etc/resolv.conf"
	dnsTimeout   = 5
)

type DNS struct {
	udpAddr          *net.UDPAddr
	udpListener      *net.UDPConn
	cnDNS            string
	fqDNS            string
	originalResolver []byte
	ipchecker        *IPChecker
}

func NewDNS(laddr, cnDNS, fqDNS string) (*DNS, error) {
	uaddr, err := net.ResolveUDPAddr("udp", laddr)
	if err != nil {
		return nil, err
	}
	ipchecker, err := NewIPChecker()
	if err != nil {
		return nil, err
	}
	return &DNS{
		udpAddr:   uaddr,
		cnDNS:     cnDNS,
		fqDNS:     fqDNS,
		ipchecker: ipchecker,
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
	// TODO add cache for dns query based on TTL
	var wg sync.WaitGroup
	var cnResp []byte
	var fqResp []byte
	dnsQuery, err := s.parse(data)
	if err != nil {
		return err
	}
	wg.Add(2)
	go func(data []byte) {
		defer wg.Done()
		var err error
		cnResp, err = s.queryCN(data)
		if err != nil {
			LOG.Warn("failed to query CN dns:", dnsQuery, err)
		}
	}(data)
	go func(data []byte) {
		defer wg.Done()
		var err error
		fqResp, err = s.queryFQ(data)
		if err != nil {
			LOG.Warn("failed to query fq dns:", dnsQuery, err)
		}
	}(data)

	wg.Wait()
	// TODO no need to wait for fq if cn response first and it's a cn ip
	dnsCN, err := s.parse(cnResp)
	if err != nil {
		return err
	}
	dnsFQ, err := s.parse(fqResp)
	if err != nil {
		return err
	}
	LOG.Debug("fq", dnsFQ)
	LOG.Debug("cn", dnsCN)
	var result []byte
	if len(dnsCN.ARecords) >= 1 && s.ipchecker.InChina(dnsCN.ARecords[0].IP) {
		// if cn dns have response and it's an cn ip, we think it's a site in China
		result = cnResp
	} else {
		// use fq dns's response for all ip outside of China
		result = fqResp
	}

	if _, err := s.udpListener.WriteToUDP(result, reqUaddr); err != nil {
		return err
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

func (s *DNS) updateResolverFile() error {
	var err error
	s.originalResolver, err = ioutil.ReadFile(resolverFile)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(resolverFile, []byte("nameserver 127.0.0.1\n"), 0644)
	if err != nil {
		return err
	}
	return nil
}

func (s *DNS) restoreResolverFile() error {
	if err := ioutil.WriteFile(resolverFile, s.originalResolver, 0644); err != nil {
		return err
	}
	return nil
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
