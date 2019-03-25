package main

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"sync"
)

const (
	dnsPort      = 53
	resolverFile = "/etc/resolv.conf"
)

type DNS struct {
	udpAddr          *net.UDPAddr
	udpListener      *net.UDPConn
	cnDNS            string
	fqDNS            string
	originalResolver []byte
}

func NewDNS(cnDNS, fqDNS string) (*DNS, error) {
	uaddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:53")
	if err != nil {
		return nil, err
	}
	return &DNS{
		udpAddr: uaddr,
		cnDNS:   cnDNS,
		fqDNS:   fqDNS,
	}, nil
}

func (s *DNS) Run() error {
	var err error
	err = s.updateResolverFile()
	if err != nil {
		return err
	}
	s.udpListener, err = net.ListenUDP("udp", s.udpAddr)
	if err != nil {
		return err
	}
	defer s.udpListener.Close()
	for {
		b := make([]byte, 1024)
		n, uaddr, err := s.udpListener.ReadFromUDP(b)
		if err != nil {
			return err
		}
		go func(uaddr *net.UDPAddr, data []byte) {
			err := s.HandleUDPData(uaddr, data)
			if err != nil {
				log.Println(err)
			}
		}(uaddr, b[:n])
	}
}

func (s *DNS) Shutdown() error {
	if err := s.restoreResolverFile(); err != nil {
		return err
	}
	return nil
}

func (s *DNS) HandleUDPData(reqUaddr *net.UDPAddr, data []byte) error {
	// TODO parse dns query data if needed
	var wg sync.WaitGroup
	var cnResp []byte
	var fqResp []byte
	wg.Add(1)
	go func(data []byte) {
		defer wg.Done()
		var err error
		cnResp, err = s.queryCN(data)
		if err != nil {
			log.Println("failed to query CN dns", err)
		}
	}(data)
	wg.Add(1)
	go func(data []byte) {
		defer wg.Done()
		var err error
		fqResp, err = s.queryFQ(data)
		if err != nil {
			log.Println("failed to query fq dns", err)
		}
	}(data)

	wg.Wait()
	// TODO ChinaDNS logic, and add to bypass ipset if it's a cn ip
	_, err := s.udpListener.WriteToUDP(fqResp, reqUaddr)
	if err != nil {
		return err
	}

	return nil
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
