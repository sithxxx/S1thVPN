// Package device — "двигатель" VPN: циклы перекачки пакетов между TUN и UDP.
package device

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/netip"
	"sync/atomic"
	"time"

	"github.com/sithxxx/S1thVPN/dataplane/internal/handshake"
	"github.com/sithxxx/S1thVPN/dataplane/internal/keys"
	"github.com/sithxxx/S1thVPN/dataplane/internal/packet"
	"github.com/sithxxx/S1thVPN/dataplane/internal/protocol"
	"github.com/sithxxx/S1thVPN/dataplane/internal/ratelimit"
	"github.com/sithxxx/S1thVPN/dataplane/internal/session"
	"github.com/sithxxx/S1thVPN/dataplane/internal/tun"
)

const bufSize = protocol.TransportHeaderSize + 65535 + protocol.TagSize

type Stats struct {
	HandshakeOK, HandshakesFailed, HandshakesRateLimited     atomic.Uint64
	DropNoPeer, DropDecrypt, DropReplay, DropSpoof, DropRate atomic.Uint64
}

type Server struct {
	priv    keys.Key
	tun     tun.Device
	conn    *net.UDPConn
	Peers   *session.Table
	hsLimit *ratelimit.PerIP
	Stats   Stats
	log     *slog.Logger
}

func NewServer(priv keys.Key, t tun.Device, conn *net.UDPConn, hsPerIPPerSec float64, log *slog.Logger) *Server {
	return &Server{
		priv: priv, tun: t, conn: conn, log: log,
		Peers:   session.NewTable(),
		hsLimit: ratelimit.NewPerIP(hsPerIPPerSec, 2*hsPerIPPerSec),
	}
}

func (s *Server) Run(ctx context.Context) error {
	errc := make(chan error, 2)
	go func() { errc <- s.inboundLoop() }()
	go func() { errc <- s.outboundLoop() }()
	go s.housekeeping(ctx)
	select {
	case <-ctx.Done():
		s.conn.Close()
		s.tun.Close()
		return nil
	case err := <-errc:
		return err
	}
}

func (s *Server) inboundLoop() error {
	buf := make([]byte, bufSize)
	for {
		n, from, err := s.conn.ReadFromUDPAddrPort(buf)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		pkt := buf[:n]
		typ, err := protocol.MessageType(pkt)
		if err != nil {
			continue
		}
		switch typ {
		case protocol.TypeHandshakeInit:
			s.handleInit(pkt, from)
		case protocol.TypeTransport:
			s.handleTransport(pkt, from)
		}
	}
}

func (s *Server) handleInit(pkt []byte, from netip.AddrPort) {
	msg, err := protocol.ParseHandshakeInit(pkt)
	if err != nil {
		return
	}
	if !s.hsLimit.Allow(from.Addr()) {
		s.Stats.HandshakesRateLimited.Add(1)
		return
	}
	r, err := handshake.Newresponder(s.priv)
	if err != nil {
		return
	}
	pub, ts, err := r.ConsumeInit(msg.Noise[:])
	if err != nil {
		s.Stats.HandshakesFailed.Add(1)
		return
	}
	peer := s.Peers.ByKey(pub)
	if peer == nil || !peer.AcceptInittimestamp(ts) {
		s.Stats.HandshakesFailed.Add(1)
		return
	}
	noiseResp, sk, err := r.CreateResponse()
	if err != nil {
		return
	}
	kp := &session.Keypair{RemoteIndex: msg.SenderIndex, Send: sk.Send, Recv: sk.Recv, Created: time.Now(), Peer: peer}
	s.Peers.Register(kp)
	s.Peers.Unregister(peer.SetNext(kp))
	peer.SetEndpoint(from)

	resp := protocol.HandshakeResponse{SenderIndex: kp.LocalIndex, ReceiverIndex: msg.SenderIndex}
	copy(resp.Noise[:], noiseResp)
	s.conn.WriteToUDPAddrPort(resp.Marshal(), from)
	s.Stats.HandshakeOK.Add(1)
}

func (s *Server) handleTransport(pkt []byte, from netip.AddrPort) {
	idx, ctr, ct, err := protocol.ParseTransport(pkt)
	if err != nil {
		return
	}
	kp := s.Peers.Lookup(idx)
	if kp == nil || kp.Expired(time.Now()) {
		s.Stats.DropNoPeer.Add(1)
		return
	}
	// Расшифровка "на месте": результат пишется поверх шифртекста.
	plain, err := kp.Recv.Decrypt(ct[:0], ctr, nil, ct)
	if err != nil {
		s.Stats.DropDecrypt.Add(1)
		return
	}
	if !kp.Replay.Accept(ctr, session.RejectAfterMessage) {
		s.Stats.DropReplay.Add(1)
		return
	}
	peer := kp.Peer
	s.Peers.Unregister(peer.Confirm(kp))
	peer.SetEndpoint(from)
	if len(plain) == 0 {
		return
	}
	// Анти-спуфинг: клиент может отправлять только от СВОЕГО туннельного IP.
	if src, ok := packet.Src(plain); !ok || src != peer.TunnelIP {
		s.Stats.DropSpoof.Add(1)
		return
	}
	if !peer.LimitRX.Allow(len(plain)) {
		s.Stats.DropRate.Add(1)
		return
	}
	peer.RXBytes.Add(uint64(len(plain)))
	s.tun.Write(plain)
}

// outboundLoop: TUN -> поиск клиента по IP назначения -> шифрование -> UDP.
func (s *Server) outboundLoop() error {
	buf := make([]byte, bufSize)
	for {
		n, err := s.tun.Read(buf[protocol.TransportHeaderSize : len(buf)-protocol.TagSize])
		if err != nil {
			if errors.Is(err, net.ErrClosed) || errors.Is(err, errClosedFile) {
				return nil
			}
			return err
		}
		pkt := buf[protocol.TransportHeaderSize : protocol.TransportHeaderSize+n]
		dst, ok := packet.Dst(pkt)
		if !ok {
			continue
		}
		peer := s.Peers.ByIP(dst)
		if peer == nil {
			s.Stats.DropNoPeer.Add(1)
			continue
		}
		kp := peer.Current()
		if kp == nil || kp.Expired(time.Now()) {
			continue
		}
		if !peer.LimitTX.Allow(n) {
			s.Stats.DropRate.Add(1)
			continue
		}
		ctr, ok := kp.NextCounter()
		if !ok {
			continue
		}
		protocol.PutTransportHeader(buf[:protocol.TransportHeaderSize], ctr, nil, pkt)
		s.conn.WriteToUDPAddrPort(out, peer.Endpoint())
		peer.TXBytes.Add(uint64(n))
	}
}

func (s *Server) housekeeping(ctx context.Context) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.hsLimit.Cleanup()
		}
	}
}
