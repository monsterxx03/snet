package cidradix

import (
	"net"
	"testing"
)

func TestCIDRadix(t *testing.T) {
	tree := NewTree()
	_, cidr, _ := net.ParseCIDR("10.1.0.0/16")
	tree.AddCIDR(cidr)
	ip := net.ParseIP("10.1.0.1")
	if !tree.Contains(ip) {
		t.Fatal("should in")
	}

	ip = net.ParseIP("10.2.0.1")
	if tree.Contains(ip) {
		t.Fatal("should not in")
	}
}
