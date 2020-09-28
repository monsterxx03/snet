package redirector

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"snet/logger"
	"snet/utils"
)

const (
	tableName = "BYPASS_SNET"
	pfDev     = "/dev/pf"
	// https://github.com/apple/darwin-xnu/blob/master/bsd/net/pfvar.h#L158
	PF_OUT = 2
	// https://github.com/apple/darwin-xnu/blob/master/bsd/net/pfvar.h#L2096
	DIOCNATLOOK = 3226747927
)

// opened /dev/pf
var pf *os.File
var pfLock *sync.Mutex = new(sync.Mutex)

// https://github.com/apple/darwin-xnu/blob/master/bsd/net/pfvar.h#L1773
type pfioc_natlook struct {
	saddr, daddr, rsaddr, rdaddr [16]byte	
	sxport, dxport, rsxport, rdxport [4]byte
	af, proto, proto_variant, direction uint8
}

type PFTable struct {
	Name        string
	bypassCidrs []string
}

func (t *PFTable) Add(ip string) {
	t.bypassCidrs = append(t.bypassCidrs, ip)
}

func (t *PFTable) CIDRS() string {
	return strings.Join(t.bypassCidrs, " ")
}

type PacketFilter struct {
	bypassTable *PFTable
	eni         string
	l           *logger.Logger
}

func (pf *PacketFilter) Init() error {
	return nil
}

func findActiveInterface(l *logger.Logger) string {
	result, err := utils.Sh("route", "get", "114.114.114.114")
	if err != nil {
		l.Errorf("failed to get active network interface, out: %s, err: %s", result, err)
		return "en0"
	}
	eni := ""
	for _, line := range strings.Split(result, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "interface") {
			r := strings.Split(line, " ")
			eni = r[1]
		}
	}
	if eni == "" {
		l.Error("failed to get active network interface, out:", result)
		return "en0"
	} else {
		return eni
	}
}

func (pf *PacketFilter) SetupRules(mode string, snetHost string, snetPort int, dnsPort int, cnDNS string) error {
	if mode != modeLocal {
		return errors.New("only support local mode")
	}
	cmd, err := utils.NamedFmt(`
echo '
table <{{ .bypassTable.Name }}> { {{ .bypassTable.CIDRS }} }
lo="lo0"
dev="{{ .eni }}"
rdr on $lo proto tcp from $dev to any port 1:65535 -> {{.snetHost }} port {{ .snetPort }}  # let proxy handle tcp 
rdr on $lo proto udp from $dev to any port 53 -> {{ .snetHost }} port {{ .dnsPort }}  # let proxy handle dns query
pass out on $dev route-to $lo proto tcp from $dev to any port 1:65535  # re-route outgoing tcp
pass out on $dev route-to $lo proto udp from $dev to any port 53  # re-route outgoing udp 
pass out proto udp from any to {{ .cnDNS }} # skip cn dns
pass out proto tcp from any to <{{ .bypassTable.Name}}>  # skip cn ip + upstream proxy ip
' | sudo pfctl -ef -
`, map[string]interface{}{"bypassTable": pf.bypassTable, "snetHost": snetHost,
		"snetPort": snetPort, "cnDNS": cnDNS, "dnsPort": dnsPort, "eni": pf.eni})
	if err != nil {
		pf.l.Error(err)
		return err
	}
	if out, err := utils.Sh(cmd); err != nil {
		pf.l.Error("output:", out, "err:", err)
		return err
	}
	return nil
}

func (pf *PacketFilter) CleanupRules(mode string, snetHost string, snetPort int, dnsPort int) error {
	if mode != modeLocal {
		return errors.New("only support local mode")
	}
	return nil
}

func (pf *PacketFilter) Destroy() {
	utils.Sh("pfctl -d")
}

func (pf *PacketFilter) ByPass(ip string) error {
	pf.bypassTable.Add(ip)
	return nil
}

func NewRedirector(byPassRoutes []string, byPassSrcIPs []string, eni string, l *logger.Logger) (Redirector, error) {
	// byPassSrcIPs is useless on mac, since it only works on router mode
	if _, err := utils.Sh("which pfctl"); err != nil {
		return nil, err
	}
	if eni == "" {
		eni = findActiveInterface(l)
	}
	l.Info("using interface ", eni)
	bypass := append(byPassRoutes, whitelistCIDR...)
	pfTable := &PFTable{Name: tableName, bypassCidrs: bypass}
	return &PacketFilter{pfTable, eni, l}, nil
}

func ioctl(fd uintptr, cmd uintptr, ptr unsafe.Pointer) error {
	if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, fd, cmd, uintptr(ptr)); err != 0 {
		return err
	}
	return nil
}

func getDevFd() (*os.File, error) {
	pfLock.Lock()
	defer pfLock.Unlock()
	if pf == nil {
		f, err := os.OpenFile(pfDev, os.O_RDWR, 0644)
		if err != nil {
			return nil, err
		}
		pf = f
	}
	return pf, nil
}

func GetDstAddr(conn *net.TCPConn) (dstHost string, dstPort int, err error) {
	caddr, ok := conn.RemoteAddr().(*net.TCPAddr)
	if !ok {
		return "", -1, errors.New("failed to get client address")
	}
	laddr, ok := conn.LocalAddr().(*net.TCPAddr)
	if !ok {
		return "", -1, errors.New("failed to get bind address")
	}
	pff, err := getDevFd()
	if err != nil {
		return "", -1, err
	}
	pffd := pff.Fd()
	pnl := new(pfioc_natlook)
	pnl.direction = PF_OUT
	pnl.af = syscall.AF_INET
	pnl.proto = syscall.IPPROTO_TCP

	// fullfill client ip & port
	copy(pnl.saddr[:4], caddr.IP)
	cport := make([]byte, 2)
	binary.BigEndian.PutUint16(cport, uint16(caddr.Port))
	copy(pnl.sxport[:], cport)

	// fullfill local proxy's bind ip & port
	copy(pnl.daddr[0:4], laddr.IP)
	lport := make([]byte, 2)
	binary.BigEndian.PutUint16(lport, uint16(laddr.Port))
	copy(pnl.dxport[:], lport)

	// do lookup
	if err := ioctl(pffd, DIOCNATLOOK, unsafe.Pointer(pnl)); err != nil {
		return "", -1, err
	}

	// get redirected ip & port
	rport := make([]byte, 2)
	copy(rport, pnl.rdxport[:2])
	raddr := pnl.rdaddr[:4]
	return fmt.Sprintf("%d.%d.%d.%d", raddr[0], raddr[1], raddr[2], raddr[3]), int(binary.BigEndian.Uint16(rport)), nil
}
