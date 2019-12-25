// Package cidradix use radix tree to store cidrs for chnroutes.
// Used to check whether a ip is in cidrs quickly, and use less memory.
package cidradix

import (
	"encoding/binary"
	"net"
)

const (
	startbit     = uint32(0x80000000)
	placeholdval = 1
)

type Node struct {
	left  *Node
	right *Node
	value int
}

type Tree struct {
	root *Node
}

func NewTree() *Tree {
	t := new(Tree)
	t.root = new(Node)
	return t
}

func (t *Tree) AddCIDR(cidr *net.IPNet) {
	bit := startbit
	node := t.root
	next := t.root
	ip := ip2uint32(cidr.IP)
	mask := ip2uint32(cidr.Mask)
	// search until we find target parent node
	for (bit & mask) > 0 {
		if (ip & bit) > 0 {
			next = node.right
		} else {
			next = node.left
		}
		// reach end
		if next == nil {
			break
		}
		bit >>= 1
		node = next
	}
	// already has this cidr, update value
	if next != nil {
		node.value = placeholdval
		return
	}
	// build left nodes
	for (bit & mask) > 0 {
		next = new(Node)
		if (ip & bit) > 0 {
			node.right = next
		} else {
			node.left = next
		}
		bit >>= 1
		node = next
	}
	node.value = placeholdval
}

func (t *Tree) Contains(ip net.IP) bool {
	bit := startbit
	node := t.root
	_ip := ip2uint32(ip)
	for node != nil {
		if node.value == placeholdval {
			// reach the bottom leaf node
			return true
		}
		if (_ip & bit) > 0 {
			node = node.right
		} else {
			node = node.left
		}
		bit >>= 1
	}
	return false
}

func ip2uint32(ip []byte) uint32 {
	if len(ip) == 16 {
		// store ipv4 in tail
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}
