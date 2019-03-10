package main

func isIPv4(pkt []byte) bool {
	return (pkt[0] >> 4) == 4
}

func isIPv6(pkt []byte) bool {
	return (pkt[0] >> 4) == 6
}

func ipv4HL(pkt []byte) uint8 {
	return (pkt[0] & 0xf) * 4
}
