// Package id — генератор UUID (v4), реализация ports.IDGenerator.
//
// Используется crypto/rand, чтобы не тянуть внешние зависимости ради UUID.
package id

import (
	"crypto/rand"
	"encoding/hex"
)

// Generator — генератор случайных UUID v4.
type Generator struct{}

// New создаёт генератор.
func New() Generator { return Generator{} }

// NewID возвращает новый UUID v4 в каноничном строковом виде.
func (Generator) NewID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand на поддерживаемых платформах не должен падать; это
		// критическая ошибка окружения.
		panic("id: не удалось прочитать crypto/rand: " + err.Error())
	}
	// Версия 4 и variant RFC 4122.
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	var buf [36]byte
	hex.Encode(buf[0:8], b[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], b[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], b[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], b[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:36], b[10:16])
	return string(buf[:])
}
