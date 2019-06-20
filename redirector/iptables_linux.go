package redirector

import (
	"fmt"
	"io"
	"log"
	"net"
	exec "os/exec"
	"strconv"
	"strings"
	"syscall"

	utils "snet/utils"
)

const (
	chainName       = "SNET"
	setName         = "BYPASS_SNET"
	SO_ORIGINAL_DST = 80 // from: /usr/include/linux/netfilter_ipv4.h
)

type IPSet struct {
	Name        string
	bypassCidrs []string
}

func (s *IPSet) Add(ip string) error {
	s.bypassCidrs = append(s.bypassCidrs, ip)
	if _, err := utils.Sh("ipset add", s.Name, ip); err != nil {
		return err
	}
	return nil
}

func (s *IPSet) Init() error {
	s.Destroy()
	result := make([]string, 0, len(s.bypassCidrs)+1)
	result = append(result, "create "+s.Name+" hash:net family inet hashsize 1024 maxelem 65536")
	for _, route := range s.bypassCidrs {
		result = append(result, "add "+s.Name+" "+route)
	}
	cmd := exec.Command("ipset", "restore")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, strings.Join(result, "\n"))
	}()
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(string(out))
		return err
	}
	return nil
}

func (s *IPSet) Destroy() {
	// ignore error, since this function will be called during starting
	utils.Sh("ipset destroy", s.Name)
}

type IPTables struct {
	ipset *IPSet
}

func (r *IPTables) Init() error {
	return r.ipset.Init()
}

func (r *IPTables) SetupRules(mode string, snetHost string, snetPort int, dnsPort int, cnDNS string) error {
	r.CleanupRules(mode, snetHost, snetPort, dnsPort)
	port := strconv.Itoa(snetPort)
	dport := strconv.Itoa(dnsPort)
	if _, err := utils.Sh("iptables -t nat -N", chainName); err != nil {
		return err
	}
	// by pass all tcp traffic for ips in BYPASS_SNET set
	if _, err := utils.Sh("iptables -t nat -A ", chainName, "-p tcp -m set --match-set", r.ipset.Name, "dst -j RETURN"); err != nil {
		return err
	}
	// redirect all tcp traffic in SNET chain to local proxy port
	if _, err := utils.Sh("iptables -t nat -A ", chainName, "-p tcp -j REDIRECT --to-ports", port); err != nil {
		return err
	}
	// send all output tcp traffic to SNET chain
	if _, err := utils.Sh("iptables -t nat -A OUTPUT -p tcp -j", chainName); err != nil {
		return err
	}
	if mode == modeLocal {
		// avoid outgoing cn dns query be redirected to snet, it's a loop!
		if _, err := utils.Sh("iptables -t nat -A", chainName, "-d", cnDNS, "-j RETURN"); err != nil {
			return err
		}
		// redirect all outgoing dns query to snet(except cn dns)
		if _, err := utils.Sh("iptables -t nat -A", chainName, "-p udp --dport 53 -j DNAT --to-destination", snetHost+":"+dport); err != nil {
			return err
		}

		if _, err := utils.Sh("iptables -t nat -A OUTPUT -p udp --dport 53 -j", chainName); err != nil {
			return err
		}
	}
	if mode == modeRouter {
		if _, err := utils.Sh("iptables -t nat -I PREROUTING -p tcp -j", chainName); err != nil {
			return err
		}
		if _, err := utils.Sh("iptables -t nat -I PREROUTING -p udp --dport 53 -j REDIRECT --to-port", dport); err != nil {
			return err
		}
	}
	return nil
}

func (r *IPTables) CleanupRules(mode string, snetHost string, snetPort int, dnsPort int) error {
	if mode != modeLocal && mode != modeRouter {
		return fmt.Errorf("Invalid mode %s", mode)
	}
	dport := strconv.Itoa(dnsPort)
	utils.Sh("iptables -t nat -D OUTPUT -p tcp -j ", chainName)
	if mode == modeLocal {
		utils.Sh("iptables -t nat -D", chainName, "-p  udp --dport 53 -j DNAT --to-destination", snetHost+":"+dport)
		utils.Sh("iptables -t nat -D OUTPUT -p udp --dport 53 -j", chainName)
	}
	if mode == modeRouter {
		utils.Sh("iptables -t nat -D PREROUTING -p tcp -j", chainName)
		utils.Sh("iptables -t nat -D PREROUTING -p udp --dport 53 -j REDIRECT --to-port", dport)
	}
	utils.Sh("iptables -t nat -F", chainName)
	utils.Sh("iptables -t nat -X", chainName)
	return nil
}

func (r *IPTables) Destroy() {
	r.ipset.Destroy()
}

func (r *IPTables) ByPass(ip string) error {
	return r.ipset.Add(ip)
}

func GetDstAddr(conn *net.TCPConn) (dstHost string, dstPort int, err error) {
	f, err := conn.File()
	if err != nil {
		return "", -1, err
	}
	// f is a copy of tcp connection's underlying fd, close it won't affect current connection
	defer f.Close()
	fd := f.Fd()
	addr, err := syscall.GetsockoptIPv6Mreq(int(fd), syscall.IPPROTO_IP, SO_ORIGINAL_DST)
	if err != nil {
		return "", -1, err
	}
	// ipv4 addr is bytes 5 to 8
	// port number is byte 2 and 3
	host := fmt.Sprintf("%d.%d.%d.%d",
		addr.Multiaddr[4],
		addr.Multiaddr[5],
		addr.Multiaddr[6],
		addr.Multiaddr[7],
	)
	return host, int(addr.Multiaddr[2])<<8 + int(addr.Multiaddr[3]), err
}

func NewRedirector(byPassRoutes []string) (Redirector, error) {

	if _, err := utils.Sh("which ipset"); err != nil {
		return nil, err
	}
	bypass := append(byPassRoutes, whitelistCIDR...)
	ipset := &IPSet{Name: setName, bypassCidrs: bypass}
	return &IPTables{ipset}, nil
}
