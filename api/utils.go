package api

import (
	"net"
	"net/netip"
)

func netIPToNetipAddr(ip net.IP) (netip.Addr, bool) {
	if ip == nil {
		return netip.Addr{}, false
	}

	if v4 := ip.To4(); v4 != nil {
		var b [4]byte
		copy(b[:], v4)
		return netip.AddrFrom4(b), true
	}

	ip16 := ip.To16()
	if ip16 == nil {
		return netip.Addr{}, false
	}
	var b [16]byte
	copy(b[:], ip16)
	return netip.AddrFrom16(b), true
}
