package ratelimit

import (
	"net/netip"
	"sync"
	"time"
)

type PerIP struct {
	mu      sync.Mutex
	rate    float64
	burst   float64
	buckets map[netip.Addr]*Bucket
}

func NewPerIP(perSecond, burst float64) *PerIP {
	return &PerIP{rate: perSecond, burst: burst, buckets: make(map[netip.Addr]*Bucket)}
}

func (l *PerIP) Allow(ip netip.Addr) bool {
	l.mu.Lock()
	b, ok := l.buckets[ip]
	if !ok {
		b = New(l.rate, l.burst)
		l.buckets[ip] = b
	}
	l.mu.Unlock()
	return b.Allow(1)
}

func (l *PerIP) Cleanup() {
	l.mu.Lock()
	for ip, b := range l.buckets {
		b.mu.Lock()
		idle := time.Since(b.last) > time.Minute
		b.mu.Unlock()
		if idle {
			delete(l.buckets, ip)
		}
	}
}
