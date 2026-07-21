package memory

import (
	"fmt"
	"sync/atomic"
)

// SeqIDs — детерминированный генератор идентификаторов для тестов.
// Выдаёт UUID-подобные строки фиксированной длины 36 символов.
type SeqIDs struct {
	n atomic.Int64
}

// NewSeqIDs создаёт генератор.
func NewSeqIDs() *SeqIDs { return &SeqIDs{} }

// NewID возвращает очередной детерминированный идентификатор.
func (s *SeqIDs) NewID() string {
	v := s.n.Add(1)
	// Формат ровно 36 символов (8-4-4-4-12), как каноничный UUID.
	return fmt.Sprintf("%08x-0000-4000-8000-%012x", v, v)
}
