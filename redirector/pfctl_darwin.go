package redirector

import (
	"fmt"
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
	bypassTable *PFTable
}

func (pf *PacketFilter) Init() error {
	return nil
}

func (pf *PacketFilter) SetupRules(mode string, snetHost string, snetPort int, dnsPort int, cnDNS string) error {
	cmd := fmt.Sprintf(`
echo "
%s
dev=en0
lo=lo0
rdr pass log on $lo inet proto tcp from $dev to any port 1:65535 -> %s port  %d
#pass out on $dev from <%s> to any
pass out on $dev route-to $lo inet proto tcp from $dev to any port 1:65535
" | sudo pfctl -ef -
`, pf.bypassTable.String(), snetHost, snetPort, pf.bypassTable.Name)
	if _, err := utils.Sh(cmd); err != nil {
		return err
	}
	return nil
}

func (pf *PacketFilter) CleanupRules(mode string, snetHost string, snetPort int, dnsPort int) error {
	if mode != modeLocal {
		return fmt.Errorf("only suport local mode")
	}
	return nil
}

func (pf *PacketFilter) Destroy() {
	utils.Sh("pfctl -d")
}

func (pf *PacketFilter) ByPass(ip string) error {
	if _, err := utils.Sh("pfctl -t", tableName, "-T add", ip); err != nil {
		return err
	}
	return nil
}

func (pf *PacketFilter) GetDstAddr() {

}

func NewRedirect(byPassRoutes []string) (Redirector, error) {
	if _, err := utils.Sh("which pfctl"); err != nil {
		return nil, err
	}
	bypass := append(byPassRoutes, whitelistCIDR...)
	pfTable := &PFTable{Name: tableName, bypassCidrs: bypass}
	return &PacketFilter{pfTable}, nil
}
