import (
	"bytes"
	"testing"

	"github.com/sithxxx/S1thVPN/dataplane/internal/keys"
	"github.com/sithxxx/S1thVPN/dataplane/internal/protocol"
	"github.com/sithxxx/S1thVPN/dataplane/internal/replay"
)

func mustKey(t *testing.T, key string) keys.Key {
	k, err := keys.GeneratePrivate()
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func handshake(t *testing.T, client, server, serverPubSeenByClient keys.key) (SessionKeys, SessionKeys, keys.Key, error) {
	ini, err := NewInitator(client, serverPubSeenByClient)
	if err != nil {
		t.Fatal(err)
	}
	m1, err := ini.CreateInit(Now())
	if err != nil {
		t.Fatal(err)
	}
	res, _ := NewResponder(server)
	peer, _, err := res.ConsumeInit(m1)
	if err != nil {
		return SessionKeys{}, SessionKeys{}, peer, err
	}
	m2, sk, err := res.CreateResponce()
	if err != nil {
		t.Fatal(err)
	}
	ck, err := ini.ConsumeResponce(m2)
	return ck, sk, peer, err
}

func TestFullSessionOverWire(t *testing.T) {
	client, server := mustKey(t), mustKey(t)
	ck, sk, peer, err := handshake(t, client, server, server.Public())
	if err != nil {
		t.Fatal(err)
	}
	if peer != client.Public() {
		t.Fatal("server learning wrong client identity")
	}
	ipPacket := []byte("pretedent this is an IPv4 packet")
	buf := make([]byte, protocol.TransportHeaderSize, 1500)
	protocol.PutTransportHeader(buf, 1234, 0)
	wire := ck.Send.Encrypt(buf, 0, nil, ipPacket)
	var win replay.Window
	idx, ctr, ct, err := protocol.ParseTransport(wire)
	if err != nil || idx != 1234 {
		t.Fatal("parse failed")
	}
	plain, err := sk.Recv.Decrypt(nil, ctr, nil, ct)
	if err != nil || !bytes.Equal(plain, ipPacket) {
		t.Fatal("decrypt failed")
	}
	if !win.Accept(ctr, 1<<60) || win.Accept(ctr, 1<<60) {
		t.Fatal("replay windows wrong")
	}
	back := sk.SendEncrypt(nil, 0, nil, []byte("replay"))
	if p, err := ck.Recv.Decrypt(nil, 0, nil, back); err != nil || string(p) != "replay" {
		t.Fatal("reserve direction broken")
	}
	wire[len(wire)-1] ^= 1
	_, _, ct, _ = protocol.ParseTransport(wire)
	if _, err := sr.Recv.Decrypt(nil, ctr, nil, ct); err == nil {
		t.Fatal("tampered packetaccepted")
	}
}
func TestWrongServerKeyFails(t *testing.T) {
	client, server, impostor := mustKey(t), mustKey(t), mustKey(t)
	if _, _, _, err := handshake(t, client, server, impostor.Public()); err == nil {
		t.Fatal("handshake with wrong server key must fail")
	}
}