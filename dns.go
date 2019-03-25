package main

import (
	"golang.org/x/net/dns/dnsmessage"

	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"sort"
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

func NewDNS(cnDNS, fqDNS string) (*DNS, error) {
	uaddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:53")
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
			err := s.handle(uaddr, data)
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

func (s *DNS) handle(reqUaddr *net.UDPAddr, data []byte) error {
	// TODO add cache for dns query based on TTL
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
	// TODO no need to wait for fq if cn response first and it's a cn ip
	cndm, cn, _ := s.extractIPs(cnResp)
	fqdm, fq, _ := s.extractIPs(fqResp)
	fmt.Println("fq resp", cndm, fq)
	fmt.Println("cn resp", fqdm, cn)
	var result []byte
	if len(cn) >= 1 && s.ipchecker.InChina(cn[0]) {
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

func (s *DNS) extractIPs(data []byte) (string, []net.IP, error) {
	// it sucks
	var p dnsmessage.Parser
	if _, err := p.Start(data); err != nil {
		return "", nil, err
	}
	if err := p.SkipAllQuestions(); err != nil {
		log.Println("failed to skip questions")
		return "", nil, err
	}
	var gotIPs []net.IP
	var name string
	for {
		h, err := p.AnswerHeader()
		if err == dnsmessage.ErrSectionDone {
			break
		}
		if err != nil {
			log.Println("fail on header", err)
			return "", nil, err
		}
		if h.Type != dnsmessage.TypeA {
			if err := p.SkipAnswer(); err != nil {
				return "", nil, err
			}
			fmt.Println("skip", h.Type)
			continue
		}
		r, err := p.AResource()
		if err != nil {
			log.Println("fai on a record", err)
			return "", nil, err
		}
		gotIPs = append(gotIPs, r.A[:])
		name = h.Name.String()
		break
	}
	sort.Slice(gotIPs, func(i, j int) bool { return gotIPs[i].String() < gotIPs[j].String() })
	return name, gotIPs, nil
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
