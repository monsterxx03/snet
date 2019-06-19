package redirector

import (
	"fmt"
	"io"
	"log"
	exec "os/exec"
	utils "snet/utils"
	"strconv"
	"strings"
)

const (
	chainName  = "SNET"
	setName    = "BYPASS_SNET"
	modeLocal  = "local"
	modeRouter = "router"
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
	utils.Sh("iptables -t nat -N", chainName)
	utils.Sh("iptables -t nat -A ", chainName, "-p tcp -m set --match-set", r.ipset.Name, "dst -j RETURN")
	utils.Sh("iptables -t nat -A ", chainName, "-p tcp -j REDIRECT --to-ports", port)
	utils.Sh("iptables -t nat -A OUTPUT -p tcp -j", chainName)
	if mode == modeLocal {
		// avoid outgoing cn dns query be redirected to snet, it's a loop!
		utils.Sh("iptables -t nat -A", chainName, "-d", cnDNS, "-j RETURN")
		// redirect all outgoing dns query to snet(except cn dns)
		utils.Sh("iptables -t nat -A", chainName, "-p udp --dport 53 -j DNAT --to-destination", snetHost+":"+dport)

		utils.Sh("iptables -t nat -A OUTPUT -p udp --dport 53 -j", chainName)
	}
	if mode == modeRouter {
		utils.Sh("iptables -t nat -I PREROUTING -p tcp -j", chainName)
		utils.Sh("iptables -t nat -I PREROUTING -p udp --dport 53 -j REDIRECT --to-port", dport)
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

func (r *IPTables) GetDstAddr() {

}

func NewRedirect(byPassRoutes []string) (Redirector, error) {

	if _, err := utils.Sh("which ipset"); err != nil {
		return nil, err
	}
	bypass := append(byPassRoutes, whitelistCIDR...)
	ipset := &IPSet{Name: setName, bypassCidrs: bypass}
	return &IPTables{ipset}, nil
}
