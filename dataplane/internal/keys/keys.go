// Package keys — ключи X25519: генерация, base64-кодирование,
// вычисление публичного ключа из приватного.
package keys

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/curve25519"
)

// Size — длина ключа X25519 в байтах.
const Size = 32

// Key — приватный или публичный ключ X25519.
type Key [Size]byte

// String кодирует ключ в base64 (так ключи хранятся в конфигах и БД).
func (k Key) String() string { return base64.StdEncoding.EncodeToString(k[:]) }

// IsZero сообщает, что ключ не задан.
func (k Key) IsZero() bool { return k == Key{} }

// Parse декодирует ключ из base64 и проверяет длину.
func Parse(s string) (Key, error) {
	var k Key
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return k, fmt.Errorf("keys: bad base64: %w", err)
	}
	if len(b) != Size {
		return k, fmt.Errorf("keys: want %d bytes, got %d", Size, len(b))
	}
	copy(k[:], b)
	return k, nil
}

// GeneratePrivate создаёт новый приватный ключ из криптостойкого ГСЧ.
func GeneratePrivate() (Key, error) {
	var k Key
	if _, err := rand.Read(k[:]); err != nil {
		return k, err
	}
	// "Clamping" по RFC 7748: делает ключ каноничным для X25519.
	k[0] &= 248
	k[31] = (k[31] & 127) | 64
	return k, nil
}

// Public вычисляет публичный ключ: X25519(private, basepoint).
func (k Key) Public() Key {
	out, err := curve25519.X25519(k[:], curve25519.Basepoint)
	if err != nil {
		panic(err) // для basepoint ошибка невозможна
	}
	var pub Key
	copy(pub[:], out)
	return pub
}
