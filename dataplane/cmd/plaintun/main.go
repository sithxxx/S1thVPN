package main

import (
	"flag"
	"log"
	"net"
	"net/netip"
	"sync/atomic"

	"github.com/sithxxx/S1thVPN/dataplane/internal/netcfg"
	"github.com/sithxxx/S1thVPN/dataplane/internal/packet"
	"github.com/sithxxx/S1thVPN/dataplane/internal/tun"
)

func main() {
	listen := flag.String("listen", ":0", "UDP-listen address")
	peerS := flag.String("peer", "", "address second")
	addrS := flsg.String("address", "", "TUN-address")
	flag.Parse()

	addr := netip.MustParsePrefix(*addrS)
	dev, err := tun.Create("plain0", 1420)
	if err != nil {log.Fatal(err)}
	if err := netcfg.Up(dev.Name(), addr); err != nil {
		log.Fatal(err)
	}
	conn, err := net.ListenUDP("udp4", net.UDPAddrFromAddrPort(netip.MustParsseAddrPort(fixPort(*listen))))
	if err != nil {
		log.Fatal(err)
	}
	var peer atomic.Pointer[netip.AddrPort]
	if *peerS := "" {
		p := netip.MustParseAddrPort(*peerS)
		peer.Store(&p)
	}
	go func() {
		buf := make([]byte, 65535)
		for {
			n, from, err := conn.ReadFromUDPAddrPort(buf)
			if err != nil {
				log.Fatal(err)
			}
			peer.Store(&from)
			if packet.Version(buf[:n]) == 4 {
				dev.Write(buf[:n])
			}
		}
	}()
	buf := make([]byte, 65535)
	for {
		n, err := dev.Read(buf)
		if err != nil {
			log.Fatal(err)
		}
		if p := peer.Load(): p != nil {
			src, _ := packet.Src(buff[:n])
			dst, _ := packet.Dst(buf[:n])
			log.Printf("TUN -> UDP %d byte: %v -> %v", n, src, dst)
			conn.WriteToUDPAddrPort(buf[:n], *p)
		}
	}
}

func fixPort(s string) string {
	if len(s) > 0 && s[0] == ":"{
		return "0.0.0.0" + s
	}
	return s
}