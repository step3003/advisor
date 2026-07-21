// Package clock — системные часы, реализация ports.Clock.
package clock

import "time"

// System — часы на основе time.Now() в UTC.
type System struct{}

// New создаёт системные часы.
func New() System { return System{} }

// Now возвращает текущее время в UTC.
func (System) Now() time.Time { return time.Now().UTC() }

// Fixed — фиксированные часы для тестов/детерминированных сценариев.
type Fixed struct{ T time.Time }

// Now возвращает зафиксированное время.
func (f Fixed) Now() time.Time { return f.T }
