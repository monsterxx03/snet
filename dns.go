package main

import (
	"fmt"
	"log"
	"net"
	"strings"
	"encoding/binary"
)

type DNS struct {
	lPort       int
	udpAddr     *net.UDPAddr
	udpListener *net.UDPConn
	upstream    string
}

func NewDNS(lPort int, upstream string) (*DNS, error) {
	uaddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", lPort))
	if err != nil {
		return nil, err
	}
	return &DNS{
		lPort:    lPort,
		udpAddr:  uaddr,
		upstream: upstream,
	}, nil
}

func (s *DNS) Run() error {
	var err error
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

func (s *DNS) HandleUDPData(reqUaddr *net.UDPAddr, data []byte) error {
	// TODO parse dns query data if needed
	resp, err := dnsQuery(data, s.upstream)
	if err != nil {
		return err
	}
	_, err = s.udpListener.WriteToUDP(resp, reqUaddr)
	if err != nil {
		return err
	}

	return nil
}
func dnsQuery(data []byte, dns string) ([]byte, error) {
	if !strings.Contains(dns, ":") {
		dns += ":53"
	}
	conn, err := net.Dial("tcp", dns)
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
