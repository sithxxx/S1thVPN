package ratelimit

import (
	"sync"
	"time"
)

type Bucket struct {
	mu     sync.Mutex
	rate   float64
	burst  float64
	tokens float64
	last   time.Time
	now    func() time.Time
}

func New(rate, burst float64) *Bucket { return newWithClock(rate, burst, time.Now) }
func newWithClock(rate, burst float64, now func() time.Time) *Bucket {
	return &Bucket{rate: rate, burst: burst, tokens: burst, last: now(), now: now}
}

func FromMbps(mbps float64) *Bucket {
	rate := mbps * 1_000_000 / 8
	burst := rate * 0.25
	if burst < 64*1024 {
		burst = 64 * 1024
	}
	return New(rate, burst)
}

func (b *Bucket) Allow(n int) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := b.now()
	b.tokens += now.Sub(b.last).Seconds() * b.rate
	if b.tokens > b.burst {
		b.tokens = b.burst
	}
	b.last = now
	if b.tokens < float64(n) {
		return false
	}
	b.tokens -= float64(n)
	return true
}

func (b *Bucket) SetRate(rate, burst float64) {
	b.mu.Lock()
	b.rate, b.burst = rate, burst
	if b.tokens > burst {
		b.tokens = burst
	}
	b.mu.Unlock()
}
