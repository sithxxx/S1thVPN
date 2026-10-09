package session

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"net/netip"
	"sync"

	"github.com/sithxxx/S1thVPN/dataplane/internal/keys"
)

var ErrIPInUse = errors.New("session: tunnel ip already used by another peer")

type Table struct {
	mu      sync.RWMutex
	byKey   map[keys.Key]*Peer
	byIP    map[netip.Addr]*Peer
	byIndex map[uint32]*Keypair
}

func NewTable() *Table {
	return &Table{
		byKey:   make(map[keys.Key]*Peer),
		byIP:    make(map[netip.Addr]*Peer),
		byIndex: make(map[uint32]*Keypair),
	}
}

func (t *Table) Upsert(pub keys.Key, ip netip.Addr, mbps int) (*Peer, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if other, ok := t.byIP[ip]; ok {
		if p.TunnelIP != ip {
			t.removeLocked(p)
		} else {
			p.SetRate(mbps)
			return p, nil
		}
	}
	p := NewPeer(pub, ip, mbps)
	t.byKey[pub], t.byIP[ip] = p, p
	return p, nil
}

func (t *Table) Remove(pub keys.Key) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if p, ok := t.byKey[pub]; ok {
		t.removeLocked(p)
	}
}

func (t *Table) removeLocked(p *Peer) {
	delete(t.byKey, p.PublicKey)
	delete(t.byIP, p.TunnelIP)
	for _, kp := range p.Keypairs() {
		delete(t.byIndex, kp.LocalIndex)
	}
}

func (t *Table) ByIP(ip netip.Addr) *Peer {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.byIP[ip]
}

func (t *Table) Lookup(index uint32) *Keypair {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.byIndex[index]
}

// Reserve резервирует случайный свободный индекс. Случайный (а не 1,2,3...),
// чтобы по индексу нельзя было угадать число клиентов. Пока индекс только
// зарезервирован, Lookup по нему возвращает nil — это безопасно.
// Клиент резервирует индекс ДО хендшейка: он уходит серверу в HandshakeInit.
func (t *Table) Reserve() uint32 {
	t.mu.Lock()
	defer t.mu.Unlock()
	var b [4]byte
	for {
		rand.Read(b[:])
		idx := binary.LittleEndian.Uint32(b[:])
		if _, busy := t.byIndex[idx]; !busy && idx != 0 {
			t.byIndex[idx] = nil
			return idx
		}
	}
}

func (t *Table) Bind(kp *Keypair) {
	t.mu.Lock()
	t.byIndex[kp.LocalIndex] = kp
	t.mu.Unlock()
}

func (t *Table) Register(kp *Keypair) {
	kp.LocalIndex = t.Reserve()
	t.Bind(kp)
}

func (t *Table) Release(idx uint32) {
	t.mu.Lock()
	delete(t.byIndex, idx)
	t.mu.Unlock()
}

func (t *Table) Unregister(kp *Keypair) {
	if kp != nil {
		t.Release(kp.LocalIndex)
	}
}

func (t *Table) Peers() []*Peer {
	t.mu.RLock()
	defer t.mu.RUnlock()
	out := make([]*Peer, 0, len(t.byKey))
	for _, p := range t.byKey {
		out = append(out, p)
	}
	return out
}
