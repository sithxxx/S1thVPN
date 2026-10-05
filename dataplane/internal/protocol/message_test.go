package protocol

import "testing"

func TestHandshakeInitRoundTrip(t *testing.T) {
	in := HandshakeInit{SenderIndex: 0xdeadbeef}
	in.Noise[0], in.MAC1[15] = 7, 9
	b := in.Marshal()
	if len(b) != HandshakeInitSize {
		t.Fatalf("size %d", len(b))
	}
	out, err := ParseHandshakeInit(b)
	if err != nil || out != in {
		t.Fatalf("roundtrip mismatch: %v", err)
	}
}

func TestTransportHeader(t *testing.T) {
	b := make([]byte, MinTransportSize)
	PutTransportHeader(b, 42, 1<<40)
	r, c, ct, err := ParseTransport(b)
	if err != nil || r != 42 || c != 1<<40 || len(ct) != TagSize {
		t.Fatalf("got %d %d %d %v", r, c, len(ct), err)
	}
}

func TestRejectsNonZeroReserved(t *testing.T) {
	b := make([]byte, MinTransportSize)
	PutTransportHeader(b, 1, 1)
	b[2] = 1
	if _, _, _, err := ParseTransport(b); err == nil {
		t.Fatal("expected error")
	}
}

// Фаззинг: парсер не должен паниковать ни на каких входных данных.
func FuzzParse(f *testing.F) {
	f.Add(make([]byte, HandshakeInitSize))
	f.Add(make([]byte, MinTransportSize))
	f.Fuzz(func(t *testing.T, b []byte) {
		ParseHandshakeInit(b)
		ParseHandshakeResponse(b)
		ParseTransport(b)
		MessageType(b)
	})
}
