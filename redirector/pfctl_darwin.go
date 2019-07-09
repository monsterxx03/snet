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

// #include <sys/ioctl.h>
// #define PRIVATE
// #include <net/pfvar.h>
// #undef PRIVATE
import "C"

const (
	tableName = "BYPASS_SNET"
	pfDev     = "/dev/pf"
)

// opened /dev/pf
var pf *os.File
var pfLock *sync.Mutex = new(sync.Mutex)

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
	l           *logger.Logger
}

func (pf *PacketFilter) Init() error {
	return nil
}

func (pf *PacketFilter) SetupRules(mode string, snetHost string, snetPort int, dnsPort int, cnDNS string) error {
	if mode != modeLocal {
		return errors.New("only support local mode")
	}
	cmd, err := utils.NamedFmt(`
echo '
table <{{ .bypassTable.Name }}> { {{ .bypassTable.CIDRS }} }
lo="lo0"
dev="en0"
rdr on $lo proto tcp from $dev to any port 1:65535 -> {{.snetHost }} port {{ .snetPort }}  # let proxy handle tcp 
rdr on $lo proto udp from $dev to any port 53 -> {{ .snetHost }} port {{ .dnsPort }}  # let proxy handle dns query
pass out on $dev route-to $lo proto tcp from $dev to any port 1:65535  # re-route outgoing tcp
pass out on $dev route-to $lo proto udp from $dev to any port 53  # re-route outgoing udp 
pass out proto udp from any to {{ .cnDNS }} # skip cn dns
pass out proto tcp from any to <{{ .bypassTable.Name}}>  # skip cn ip + upstream proxy ip
' | sudo pfctl -ef -
`, map[string]interface{}{"bypassTable": pf.bypassTable, "snetHost": snetHost,
		"snetPort": snetPort, "cnDNS": cnDNS, "dnsPort": dnsPort})
	if err != nil {
		pf.l.Error(err)
		return err
	}
	if _, err := utils.Sh(cmd); err != nil {
		pf.l.Error(err)
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

func NewRedirector(byPassRoutes []string, l *logger.Logger) (Redirector, error) {
	if _, err := utils.Sh("which pfctl"); err != nil {
		return nil, err
	}
	bypass := append(byPassRoutes, whitelistCIDR...)
	pfTable := &PFTable{Name: tableName, bypassCidrs: bypass}
	return &PacketFilter{pfTable, l}, nil
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
	pnl := new(C.struct_pfioc_natlook)
	pnl.direction = C.PF_OUT
	pnl.af = C.AF_INET
	pnl.proto = C.IPPROTO_TCP

	// fullfill client ip & port
	copy(pnl.saddr.pfa[:4], caddr.IP)
	cport := make([]byte, 2)
	binary.BigEndian.PutUint16(cport, uint16(caddr.Port))
	copy(pnl.sxport[:], cport)

	// fullfill local proxy's bind ip & port
	copy(pnl.daddr.pfa[0:4], laddr.IP)
	lport := make([]byte, 2)
	binary.BigEndian.PutUint16(lport, uint16(laddr.Port))
	copy(pnl.dxport[:], lport)

	// do lookup
	if err := ioctl(pffd, C.DIOCNATLOOK, unsafe.Pointer(pnl)); err != nil {
		return "", -1, err
	}

	// get redirected ip & port
	rport := make([]byte, 2)
	copy(rport, pnl.rdxport[:2])
	raddr := pnl.rdaddr.pfa[:4]
	return fmt.Sprintf("%d.%d.%d.%d", raddr[0], raddr[1], raddr[2], raddr[3]), int(binary.BigEndian.Uint16(rport)), nil
}
