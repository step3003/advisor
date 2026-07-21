package money

import "errors"

// ErrBadRate — некорректные параметры курса.
var ErrBadRate = errors.New("money: некорректный курс")

// Convert пересчитывает сумму в целевую валюту по дробному курсу num/den.
//
// Результат в минорных единицах: target_minor = round( m.minor * num / den ),
// где округление — «половина вверх» по модулю (банковский half-up для положительных
// и отрицательных значений симметрично). Вся арифметика целочисленная — без float.
//
// Для курса НБ РБ (Cur_Scale единиц валюты = Cur_OfficialRate BYN):
//
//	num = rateBYNMinor          // официальный курс в минорных единицах BYN
//	den = 100 * scale           // 100 — перевод BYN-major в minor, scale — Cur_Scale
func Convert(m Money, target Currency, num, den int64) (Money, error) {
	if target.IsZero() {
		return Money{}, ErrEmptyCurrency
	}
	if den <= 0 || num < 0 {
		return Money{}, ErrBadRate
	}

	result := mulDivRoundHalfUp(m.minor, num, den)
	return Money{minor: result, currency: target}, nil
}

// mulDivRoundHalfUp вычисляет round(a*b/den) целочисленно с округлением половины вверх.
// Знак сохраняется симметрично: округление выполняется по модулю.
func mulDivRoundHalfUp(a, b, den int64) int64 {
	neg := (a < 0)
	if a < 0 {
		a = -a
	}
	// b и den уже неотрицательны/положительны по контракту Convert.
	product := a * b
	q := product / den
	rem := product % den
	// half-up: если удвоенный остаток >= делителя — округляем вверх.
	if rem*2 >= den {
		q++
	}
	if neg {
		q = -q
	}
	return q
}
