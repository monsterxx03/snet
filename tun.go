package main

import (
	"fmt"
	"github.com/songgao/water"
	"log"
	"os/exec"
)

var TUN_NAME = "tun100"

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
	cmd := exec.Command("sh", "-c", fmt.Sprintf("ip addr add %s  dev %s", t.Addr, t.Ifce.Name()))
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Println(string(out))
		return err
	}
	log.Printf("Setup tun device: %s with ip %s\n", t.Ifce.Name(), t.Addr)
	return nil
}

func (t *Tun) Read() error {
	packet := make([]byte, 1024)
	for {
		n, err := t.Ifce.Read(packet)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Packet Received: %x\n", packet[:n])
	}
}

func (t *Tun) TearDown() error {
	return nil
}
