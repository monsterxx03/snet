package main

import (
	"github.com/songgao/water"
	"log"
)

var TUN_NAME = "tun100"
var MTU = 1500

type Tun struct {
	Addr string
	Ifce *water.Interface
}

func NewTun(addr string) *Tun {
	return &Tun{Addr: addr}
}

// Setup bring up tun device and assign an ip address.
func (t *Tun) Setup() error {
	var err error
	t.Ifce, err = water.New(water.Config{
		DeviceType: water.TUN,
	})
	if err != nil {
		return err
	}
	devName := t.Ifce.Name()
	if err := Exec("ip addr add %s/24 dev %s", t.Addr, devName); err != nil {
		return err
	}
	if err := Exec("ip link set dev %s up", devName); err != nil {
		return err
	}
	if err := Exec("ip link set dev %s mtu %d", devName, MTU); err != nil {
		return err
	}
	log.Printf("Setup tun device: %s with ip %s\n", devName, t.Addr)
	return nil
}

func (t *Tun) Read() error {
	b := make([]byte, MTU)
	for {
		n, err := t.Ifce.Read(b[:])
		if err != nil {
			log.Fatal(err)
		}
		pkt := b[:n]
		if 6 == (pkt[0] >> 4) {
			log.Println("Discard ipv6 packet")
			continue
		}
		log.Println("ip header length", ipv4HL(pkt))
	}
}

func (t *Tun) TearDown() error {
	return nil
}
