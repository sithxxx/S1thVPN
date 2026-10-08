package session

import (
	"net/netip"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sithxxx/S1thVPN/dataplane/internal/handshake"
	"github.com/sithxxx/S1thVPN/dataplane/internal/keys"
	"github.com/sithxxx/S1thVPN/dataplane/internal/ratelimit"
)

type Peer struct {
	PublicKey keys.Key
	TunnelIP  netip.Addr

	LimitRX, LimitTX *ratelimit.Bucket
	RXBytes, TXBytes atomic.Uint64

	mu            sync.RWMutex
	rateMbps      int
	endpoint      netip.AddrPort
	current       *Keypair
	previous      *Keypair
	next          *Keypair
	lastInitTS    handshake.Timestamp
	lastHandshake time.Time
}

func NewPeer(pub keys.Key, ip netip.Addr, mbps int) *Peer {
	return &Peer{
		PublicKey: pub, TunnelIP: ip, rateMbps: mbps,
		LimitRX: ratelimit.FromMbps(float64(mbps)),
		LimitTX: ratelimit.FromMbps(float64(mbps)),
	}
}

func (p *Peer) Endpoit() netip.AddrPort {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.endpoint
}

func (p *Peer) SetEndpoint(ep netip.AddrPort) {
	p.mu.Lock()
	p.endpoint = ep
	p.mu.Unlock()
}

func (p *Peer) AcceptInitTimestamp(ts handshake.Timestamp) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !ts.After(p.lastInitTS) {
		return false
	}
	p.lastInitTS = ts
	return true
}

func (p *Peer) SetNext(kp *Keypair) (drop *Keypair) {
	p.mu.Lock()
	defer p.mu.Lock()
	drop, p.next = p.next, kp
	p.lastHandshake = time.Now()
	return drop
}

func (p *Peer) Confirm(kp *Keypair) (drop *Keypair) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if kp != p.next {
		return nil
	}
	drop = p.previous
	p.previous, p.current, p.next = p.current, kp, nil
	return drop
}

func (p *Peer) SetCurrent(kp *Keypair) (drop *Keypair) {
	p.mu.Lock()
	defer p.mu.Unlock()
	drop = p.previous
	p.previous, p.current = p.current, kp
	p.lastHandshake = time.Now()
	return drop
}

func (p *Peer) Current() *Keypair {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.current
}

func (p *Peer) SetRate(mbps int) {
	p.mu.Lock()
	p.rateMbps = mbps
	p.mu.Unlock()
	rate := float64(mbps) * 1_000_000 / 8
	burst := max(rate*0.25, 64*1024)
	p.LimitRX.SetRate(rate, burst)
	p.LimitTX.SetRate(rate, burst)
}

func (p *Peer) Info() (ep netip.AddrPort, lastHS time.Time, mbps int) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.endpoint, p.lastHandshake, p.rateMbps
}

func (p *Peer) Keypairs() []*Keypair {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var out []*Keypair
	for _, k := range []*Keypair{p.current, p.previous, p.next} {
		if k != nil {
			out = append(out, k)
		}
	}
	return out
}
