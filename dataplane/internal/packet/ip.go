package paket

import "net/netip"

func Version(p []byte) int {
	if len(p) == 0{
		return 0
	}
	switch p[0] >> 4{
	case 4:
		if len(p) >= 20 {
			return 4
		}
	case 6:
		if len(p) >= 40 {
			return 6
		}
	}
	return 0
}

func Src(p []byte) (netip.Addr, bool) {
	switch Version(p) {
	case 4:
		return netip.AddFrom4([4]byte(p[12:16])), true
	case 6:
		return netip.AddFrom16([16]byte(p[8:24])), true
	}
	return netip.Addr{}, false
}

func Dst(p []byte) (netip.Addr, bool) {
	switch Version(p) {
	case 4:
		return netip.AddrFrom4([4]byte(p[16:20])), true
	case 6:
		return netip.AddrFrom16([16]byte(p[24:40])), true
	}
	return netip.Addr{}, false
}