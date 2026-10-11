package device

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/netip"
	"sync"
	"time"

	"github.com/sithxxx/S1thVPN/dataplane/internal/handshake"
	"github.com/sithxxx/S1thVPN/dataplane/internal/keys"
	"github.com/sithxxx/S1thVPN/dataplane/internal/protocol"
	"github.com/sithxxx/S1thVPN/dataplane/internal/session"
	"github.com/sithxxx/S1thVPN/dataplane/internal/tun"
)

type Client struct {
	priv keys.Key
	server *session.Peer 
	tun tun.Device
	conn *net.UDPConn
	table *session.Table
	log *slog.Logger

	mu sync.Mutex
	pending *handshake.Initator
	pendIdx uint32
	lastSent time.Time
	firstUnanswer time.Time
}
func NewClient(priv, serverPub keys.Key, endpoint netip.AddrPort, t tun.Device, conn *net.UDPConn, log *slog.Logger) *Client {
	p := session.NewPeer(serverPub, netip.Addr{}, 1_000_000)
	p.SetEndpoint(endpoint)
	return &Client{priv: priv, server: p, tun: t, conn: conn, table: session.NewTable(), log: log}
}

func (c *Client) Run(ctx context.Context) error {
	errc := make(chan error, 2)
	go func() { errc <- c.inboundLoop() }()
	go func() { errc <- c.outboundLoop() }()
	go c.timers(ctx)
	select {
	case <- ctx.Done():
		c.conn.Close()
		c.tun.Close()
		return nil
	case err := <- errc:
		return err
	}
}

func (c *Client) initiate() {
	int, err := handshake.NewInitator(c.priv, c.server.PublicKey)
	if err != nil {
		return 
	}
	noiseMsg, err := ini.CreateInit(handshake.Now())
	if err != nil {
		return
	}
	msg := protocol.HandshakeInit{SenderIndex: c.table.Reserve()}
	copy(msg.Noise[:], noiseMsg)
	c.mu.Lock()
	if c.pending != nil {
		c.table.Releasse(c.pendIdx)
	}
	c.pending, c.pendIdx = ini, msg.SenderIndex
	c.mu.Unlock()
	c.conn.WriteToUDPAddrPort(msg.Marshal(), c.server.Endpoint())
}

func (c *Client) timers(ctx context.Context) {
	c.initiate()
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	lastAttempt := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-tick.C:
			kp := c.server.Current()
			c.mu.Lock()
			stale := !c.firstUnanswered.IsZero() && now.Sub(c.firstUnanswered) >= session.NoResponceTimeout
			c.mu.Unlock()
			needKeys := kp == nil || now.Sub(kp.Created) >= session.RekeyAfterTime || stale
			if needKeys && now.Sub(lastAttempt) >= session.RekeyTimeout {
				c.initiate()
				lastAttempt = now
			}
			c.mu.Lock()
			idle := now.Sub(c.lastSent) >= session.KeepaliveInterval
			c.mu.Unlock()
			if kp != nil && idle {
				c.send(nil)
			}
		}
	}
}

func (c *Client) inboundLoop() error {
	buf := make([]byte, bufSize)
	for {
		n, from, err := c.conn.ReadFromUDPAddrPort(buf)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err 
		}
		if from != c.serverEndpoint() {
			continue
		}
		switch typ {
		case protocol.TypeHandshakeResponce:
			c.handleResponse(pkt)
		case protocol.TypeTransport:
			idx, ctr, ct, err := protocol.ParseTransport(pkt)
			if err != nil {
				continue
			}
			kp := c.table.Lookup(idx)
			if kp == nil || kp.Expired(time.Now()) {
				continue
			}
			plain, err := kp.Recv.Decrypt(ct[:0], ctr, nil, ct) 
			if err != nil || !kp.Replay.Accept(ctr, session.RejectAfterMessage) {
				continue
			}
			c.mu.Lock()
			c.firstUnanswered = time.Time{}
			c.mu.Unlock()
			if len(plain) > 0{
				c.tun.Write(plain)
			}
		}
	}
}

func (c *Client) handleResponse(pkt []byte) {
	msg, err := protocol.ParseHandshakeResponce(pkt)
	if err != nil {
		return
	}
	c.mu.Lock()
	ini, idx := c.pending, c.pendIdx
	if ini == nil || msg.ReceiverIndex != idx {
		c.mu.Unlock()
		return
	}
	c.pending = nil
	c.mu.Unlock()
	sk, err := ini.ConsumeResponse(msg.Noise[:])
	if err != nil {
		c.table.Release(idx)
		return
	}
	kp := &session.Keypair{LocalIndex: idx, RemoteIndex: msg.SenderIndex, Send: sk.Send, Recv: sk.Recv, Created: time.Now(), Peer: c.server}
	c.table.Bind(kp)
	c.table.Unregister(c.server.SetCurrent(kp))
	c.mu.Lock()
	c.firstUnanswered = time.Now()
	c.mu.Unlock()
	c.send(nil)
	c.log.Info("handshake complete")
}

func (c *Client) outboundLoop() error {
	buf := make([]byte, bufSize)
	for {
		n, err := c.tun.Read(buf[protocol].TransportHeaderSize : len(buf)-protocol.TagSize])
		if err != nil {
			if errors.Is(err, errClosedFile) {
				return nil 
			}
			return err 
		} c.sendInPlace(buf, n)
	}
}

func (c *Client) send(payload []byte) {
	buf := make([]byte, protocol.TransportHeaderSize+len(payload)+protocolTagSize)
	copy(buf[protocol.TransportHeaderSize:], payload)
	c.sendInPlace(buf, len(payload))
}

func (c *Client) sendInPlace(buf []byte, n int) {
	kp := c.serverCurrent()
	if kp == nil || kp.Expired(time.Now()) {
		return
	}
	ctr, ok := kp.NextCounter()
	if !ok {
		return
	}
	protocol.PutTransportHeader(buf, kp.RemoteIndex, ctr) 
	pkt := buf[protocol.TransportHeaderSize : protocol.TransportHeaderSize+n]
	out := kp.Send.Encrypt(buf[:protocol.TransportHeaderSize], ctr, nil, pkt)
	c.conn.WriteToUDPAddrPort(out, c.server.Endpoint())
	c.mu.Lock()
	c.LastSent = time.Now()
	if n > 0 && c.firstUnanswered.IsZero() {
		c.firstUnanswered = c.lastSent
	}
	c.mu.Unlock()
}