package protocol

import (
	"encoding/binary"
	"errors"
)

const (
	TypeHandshakeInit     uint8 = 1
	TypeHandshakeResponce uint8 = 2
	TypeCookieReply       uint8 = 3
	TypeTransport         uint8 = 4
)

const (
	InitNoiseSize = 108
	RespNoiseSize = 48
	MACSize       = 16

	HandshakeInitSize     = 4 + 4 + InitNoiseSize + MACSize
	HandshakeResponceSize = 4 + 4 + 4 + RespNoiseSize + MACSize

	TransportHeaderSize = 16
	TagSize             = 16
	MinTransportSize    = TransportHeaderSize + TagSize
)

var ErrMalformed = errors.New("protocol: malformed message")

func MessageType(b []byte) (uint8, error) {
	if len(b) < 4 || b[1] != 0 || b[2] != 0 || b[3] != 0 {
		return 0, ErrMalformed
	}
	return b[0], nil
}

type HandshakeInit struct {
	SenderIndex uint32
	Noise       [InitNoiseSize]byte
	MAC1        [MACSize]byte
}

func (m *HandshakeInit) Marshal() []byte {
	b := make([]byte, HandshakeInitSize)
	b[0] = TypeHandshakeInit
	binary.LittleEndian.PutUint32(b[4:8], m.SenderIndex)
	copy(b[8:8+InitNoiseSize], m.Noise[:])
	copy(b[8+InitNoiseSize:], m.MAC1[:])
	return b
}

func ParseHandshakeInit(b []byte) (HandshakeInit, error) {
	var m HandshakeInit
	if len(b) != HandshakeInitSize || b[0] != TypeHandshakeInit {
		return m, ErrMalformed
	}
	if _, err := MessageType(b); err != nil {
		return m, err
	}
	m.SenderIndex = binary.LittleEndian.Uint32(b[4:8])
	copy(m.Noise[:], b[8:8+InitNoiseSize])
	copy(m.MAC1[:], b[8+InitNoiseSize:])
	return m, nil
}

type HandshakeResponse struct {
	SenderIndex   uint32
	ReseiverIndex uint32
	Noise         [RespNoiseSize]byte
	MAC1          [MACSize]byte
}

func (m *HandshakeResponse) Marshal() []byte {
	b := make([]byte, HandshakeResponceSize)
	b[0] = TypeHandshakeResponce
	binary.LittleEndian.PutUint32(b[4:8], m.SenderIndex)
	binary.LittleEndian.PutUint32(b[8:12], m.ReseiverIndex)
	copy(b[12:12+RespNoiseSize], m.Noise[:])
	copy(b[12+RespNoiseSize:], m.MAC1[:])
	return b
}

func ParseHandshakeResponse(b []byte) (HandshakeResponse, error) {
	var m HandshakeResponse
	if len(b) != HandshakeResponceSize || b[0] != TypeHandshakeResponce {
		return m, ErrMalformed
	}
	if _, err := MessageType(b); err != nil {
		return m, err
	}
	m.SenderIndex = binary.LittleEndian.Uint32(b[4:8])
	m.ReseiverIndex = binary.LittleEndian.Uint32(b[8:12])
	copy(m.Noise[:], b[12:12+RespNoiseSize])
	copy(m.MAC1[:], b[12+RespNoiseSize:])
	return m, nil
}

func PutTransportHeader(b []byte, receiver uint32, counter uint64) {
	_ = b[TransportHeaderSize-1]
	b[0] = TypeTransport
	b[1], b[2], b[3] = 0, 0, 0
	binary.LittleEndian.PutUint32(b[4:8], receiver)
	binary.LittleEndian.PutUint64(b[8:16], counter)
}

func ParseTransport(b []byte) (receiver uint32, counter uint64, ciphertext []byte, err error) {
	if len(b) < MinTransportSize || b[0] != TypeTransport {
		return 0, 0, nil, ErrMalformed
	}
	if _, err := MessageType(b); err != nil {
		return 0, 0, nil, err
	}
	receiver = binary.LittleEndian.Uint32(b[4:8])
	counter = binary.LittleEndian.Uint64(b[8:16])
	return receiver, counter, b[TransportHeaderSize:], nil
}
