package redirector

import (
	"snet/utils"
	"strings"
)

const (
	tableName = "BYPASS_SNET"
)

type PFTable struct {
	Name        string
	bypassCidrs []string
}

func (t *PFTable) String() string {
	return fmt.Sprintf("table <%s> { %s }", t.Name, strings.Join(t.bypassCidrs, " "))
}

type PacketFilter struct {
	bypassTable *pfTable
}

func (pf *PacketFilter) Init() error {
	return nil
}

func (pf *PacketFilter) SetupRules(mode string, snetHost string, snetPort int, dnsPort int, cnDNS string) error {
	rule := fmt.Sprintf(`
%s
dev=en0
lo=lo0
rdr pass log on $lo inet proto tcp from $dev to any port 1:65535 -> %s port  %d
pass out on $dev from <%s> to any
pass out on $dev route-to $lo inet proto tcp from $dev to any port 1:65525
`, pf.bypassTable.String(), pf.bypassTable.Name, snetHost, snetPort)
	fmt.Println(rule)
}

func (pf *PacketFilter) CleanupRules(mode string, snetHost string, snetPort int, dnsPort int) error {
	if mode != modelLocal {
		return fmt.Errorf("only suport local mode")
	}
}

func (pf *PacketFilter) Destroy() {
	utils.Sh("pfctl -d")
}

func (r *IPTables) ByPass(ip string) error {
	if _, err := utils.Sh("pfctl -t", tableName, "-T add", ip); err != nil {
		return err
	}
	return nil
}

func (pf *PacketFilter) GetDstAddr() {

}

func NewRedirect(byPassRoutes []string) (Redirector, error) {
	if _, err := utils.Sh("pfctl"); err != nil {
		return nil, err
	}
	bypass := append(byPassRoutes, whitelistCIDR...)
	pfTable := &PFTable{Name: tableName, bypassCidrs: bypass}
	return &PacketFilter{pfTable}, nil
}
