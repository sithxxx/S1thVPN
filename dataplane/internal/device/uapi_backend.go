package device

import (
	"fmt"
	"net/netip"

	"github.com/sithxxx/S1thVPN/dataplane/internal/keys"
	"github.com/sithxxx/S1thVPN/dataplane/internal/uapi"
)

type UAPIBackend struct {
	S      *Server
	Subnet netip.Prefix
}

func (b UAPIBackend) SetPeer(pc uapi.PeerConfig) error {
	pub, err := keys.Parse(pc.PublicKey)
	if err != nil {
		return fmt.Errorf("bad_public_key")
	}
	ip, err := netip.ParseAddr(pc.TunnelIP)
	if err != nil || !b.Subnet.Contains(ip) || ip == b.Subnet.Addr().Next() {
		return fmt.Errorf("bad_tunnel_ip")
	}
	if pc.RateLimitMbps <= 0 || pc.RateLimitMbps > 10000 {
		return fmt.Errorf("bad_rate_limit")
	}
	if _, err := b.S.Peers.Upsert(pub, ip, pc.RateLimitMbps); err != nil {
		return fmt.Errorf("tunnel_ip_in_use")
	}
	return nil
}

func (b UAPIBackend) RemovePeer(k string) error {
	pub, err := keys.Parse(k)
	if err != nil {
		return fmt.Errorf("bad_public_key")
	}
	b.S.Peers.Remove(pub)
	return nil
}

func (b UAPIBackend) ListPeers() []uapi.PeerInfo {
	var out []uapi.PeerInfo
	for _, p := range b.S.Peers.Peers() {
		ep, hs, mbps := p.Info()
		info := uapi.PeerInfo{
			PeerConfig: uapi.PeerConfig{PublicKey: p.PublicKey.String(),
				TunnelIP: p.TunnelIP.String(), RateLimitMbps: mbps},
			RXBytes: p.RXBytes.Load(), TXBytes: p.TXBytes.Load(),
		}
		if ep.IsValid() {
			info.Endpoint = ep.String()
		}
		if !hs.IsZero() {
			info.LastHandshakeUnix = hs.Unix()
		}
		out = append(out, info)
	}
	return out
}
