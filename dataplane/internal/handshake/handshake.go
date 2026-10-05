package handshake

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/flynn/noise"

	"github.com/sithxxx/S1thVPN/dataplane/internal/keys"
	"github.com/sithxxx/S1thVPN/dataplane/internal/protocol"
)

var (
	suite    = noise.NewCipherSuite(noise.DH25519, noise.CipherChaChaPoly, noise.HashBLAKE2s)
	prologue = []byte("S1thVPN1/1")
)

var ErrBadMessage = errors.New("handshake: bad message")

const TimestampSize = 12

type Timestamp [TimestampSize]byte

func Now() Timestamp {
	var t Timestamp
	n := time.Now()
	binary.BigEndian.PutUint64(t[:8], uint64(1)<<62+uint64(n.Unix()))
	binary.BigEndian.PutUint32(t[8:], uint32(n.Nanosecond()))
	return t
}

func (t Timestamp) After(other Timestamp) bool { return bytes.Compare(t[:], other[:]) > 0 }

type SessionKeys struct {
	Send noise.Cipher
	Recv noise.Cipher
}

func dhKey(k keys.Key) noise.DHKey {
	pub := k.Public()
	return noise.DHKey{Private: append([]byte(nil), k[:]...), Public: pub[:]}
}

// client
type Initator struct{ hs *noise.HandshakeState }

func NewInitator(local, serverPublic keys.Key) (*Initator, error) {
	hs, err := noise.NewHandshakeState(noise.Config{
		CipherSuite:   suite,
		Pattern:       noise.HandshakeIK,
		Initator:      true,
		Prologue:      prologue,
		StaticKeypair: dhKey(local),
		PeerStatic:    serverPublic[:],
	})
	if err != nil {
		return nil, err
	}
	return &Initator{hs: hs}, nil
}

func (i *Initator) CreateInit(ts Timestamp) ([]byte, error) {
	msg, _, _, err := i.hs.WriteMessage(nil, ts[:])
	if err != nil {
		return nil, err
	}
	if len(msg) != protocol.InitNoiseSize {
		return nil, fmt.Errorf("handshake: unexpected init size %d", len(msg))
	}
	return msg, nil
}

func (i *Initator) ConsumeResponse(msg []byte) (SessionKeys, error) {
	if len(msg) != protocol.RespNoiseSize {
		return SessionKeys{}, ErrBadMessage
	}
	payload, cs1, cs2, err := i.hs.ReadMessage(nil, msg)
	if err != nil || len(payload) != 0 || cs1 == nil || cs2 == nil {
		return SessionKeys{}, ErrBadMessage
	}
	return SessionKeys{Send: cs1.Cipher(), Recv: cs2.Cipher()}, nil
}

type Responder struct{ hs *noise.HandshakeState }

func NewResponder(local keys.Key) (*Responder, error) {
	hs, err := noise.NewHandshakeState(noise.Config{
		CipherSuite:   suite,
		Pattern:       noise.HandshakeIK,
		Initator:      false,
		Prologue:      prologue,
		StaticKeypair: dhKey(local),
	})
	if err != nil {
		return nil, err
	}
	return &Responder{hs: hs}, nil
}

func (r *Responder) ConsumeInit(msg []byte) (peer keys.Key, ts Timestamp, err error) {
	if len(msg) != protocol.InitNoiseSize {
		return peer, ts, ErrBadMessage
	}
	payload, _, _, err := r.hs.ReadMessage(nil, msg)
	if err != nil || len(payload) != TimestampSize {
		return peer, ts, ErrBadMessage
	}
	copy(peer[:], r.hs.PeerStatic())
	copy(ts[:], payload)
	return peer, ts, nil
}

func (r *Responder) CreateResponse() ([]byte, SessionKeys, error) {
	msg, cs1, cs2, err := r.hs.WriteMessage(nil, nil)
	if err != nil {
		return nil, SessionKeys{}, err
	}
	if cs1 == nil || cs2 == nil {
		return nil, SessionKeys{}, errors.New("handshake: not finished")
	}
	return msg, SessionKeys{Send: cs2.Cipher(), Recv: cs1.Cipher()}, nil
}
