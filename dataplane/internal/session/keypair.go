package session

import (
	"sync/atomic"
	"time"

	"github.com/flynn/noise"
	"github.com/sithxxx/S1thVPN/dataplane/internal/replay"
)

const (
	RekeyAfterTime     = 120 * time.Second
	RejectAfterTime    = 180 * time.Second
	RekeyTimeout       = 5 * time.Second
	KeepaliveInterval  = 25 * time.Second
	NoResponseTimeout  = 15 * time.Second
	RejectAfterMessage = uint64(1) << 60
)

type Keypair struct {
	LocalIndex  uint32
	RemoteIndex uint32
	Send, Recv  noise.Cipher
	Created     time.Time
	Replay      replay.Window
	Peer        *Peer

	sendCounter atomic.Uint64
}

func (k *Keypair) NextCounter() (uint64, bool) {
	n := k.sendCounter.Add(1) - 1
	return n, n < RejectAfterMessage
}

func (k *Keypair) Expired(now time.Time) bool { return now.Sub(k.Created) >= RejectAfterTime }
