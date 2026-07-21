package money

import "testing"

func TestParseAndDecimal(t *testing.T) {
	cases := []struct {
		in        string
		wantMinor int64
		wantDec   string
	}{
		{"42.50", 4250, "42.50"},
		{"42.5", 4250, "42.50"},
		{"42", 4200, "42.00"},
		{"0", 0, "0.00"},
		{"0.05", 5, "0.05"},
		{".5", 50, "0.50"},
		{"-3", -300, "-3.00"},
		{"-0.01", -1, "-0.01"},
		{"+7.25", 725, "7.25"},
		{"1000000.99", 100000099, "1000000.99"},
	}
	for _, c := range cases {
		m, err := Parse(c.in, "USD")
		if err != nil {
			t.Fatalf("Parse(%q) error: %v", c.in, err)
		}
		if m.Minor() != c.wantMinor {
			t.Errorf("Parse(%q).Minor = %d, want %d", c.in, m.Minor(), c.wantMinor)
		}
		if got := m.Decimal(); got != c.wantDec {
			t.Errorf("Parse(%q).Decimal = %q, want %q", c.in, got, c.wantDec)
		}
	}
}

func TestParseErrors(t *testing.T) {
	bad := []string{"", "abc", "1.234", "1..2", "1.2.3", "1 000", "12x"}
	for _, s := range bad {
		if _, err := Parse(s, "USD"); err == nil {
			t.Errorf("Parse(%q) expected error, got nil", s)
		}
	}
	if _, err := Parse("1.00", ""); err == nil {
		t.Error("Parse with empty currency should error")
	}
}

func TestArithmetic(t *testing.T) {
	a := MustNew(4250, "USD")
	b := MustNew(1750, "USD")

	sum, err := a.Add(b)
	if err != nil || sum.Minor() != 6000 {
		t.Fatalf("Add = %v, %v", sum, err)
	}
	diff, err := a.Sub(b)
	if err != nil || diff.Minor() != 2500 {
		t.Fatalf("Sub = %v, %v", diff, err)
	}
	if a.Neg().Minor() != -4250 {
		t.Errorf("Neg failed")
	}
	if a.Neg().Abs().Minor() != 4250 {
		t.Errorf("Abs failed")
	}
}

func TestCurrencyMismatch(t *testing.T) {
	a := MustNew(100, "USD")
	b := MustNew(100, "EUR")
	if _, err := a.Add(b); err == nil {
		t.Error("Add across currencies must error")
	}
	if _, err := a.Sub(b); err == nil {
		t.Error("Sub across currencies must error")
	}
	if _, err := a.Cmp(b); err == nil {
		t.Error("Cmp across currencies must error")
	}
}

func TestCmpAndEqual(t *testing.T) {
	a := MustNew(100, "BYN")
	b := MustNew(200, "BYN")
	if c, _ := a.Cmp(b); c != -1 {
		t.Errorf("Cmp(a,b) = %d, want -1", c)
	}
	if c, _ := b.Cmp(a); c != 1 {
		t.Errorf("Cmp(b,a) = %d, want 1", c)
	}
	if c, _ := a.Cmp(MustNew(100, "BYN")); c != 0 {
		t.Errorf("Cmp equal = %d, want 0", c)
	}
	if !a.Equal(MustNew(100, "BYN")) {
		t.Error("Equal failed")
	}
	if a.Equal(MustNew(100, "USD")) {
		t.Error("Equal must consider currency")
	}
}

func TestConvertNBRB(t *testing.T) {
	// Курс: 1 USD = 3.25 BYN (Cur_Scale=1, Cur_OfficialRate=3.25 => rateBYNMinor=325).
	// 42.50 USD => 42.50 * 3.25 = 138.125 BYN => округление half-up => 138.13.
	usd := MustNew(4250, "USD")
	num := int64(325) // rateBYNMinor
	den := int64(100 * 1)
	byn, err := Convert(usd, "BYN", num, den)
	if err != nil {
		t.Fatal(err)
	}
	if byn.Currency() != "BYN" {
		t.Errorf("currency = %s", byn.Currency())
	}
	if byn.Minor() != 13813 {
		t.Errorf("Convert = %d minor (%s), want 13813", byn.Minor(), byn.Decimal())
	}
}

func TestConvertWithScale(t *testing.T) {
	// Курс: 100 RUB = 4.10 BYN (Cur_Scale=100, Cur_OfficialRate=4.10 => rateBYNMinor=410).
	// 5000.00 RUB => 5000 * 4.10 / 100 = 205.00 BYN.
	rub := MustNew(500000, "RUB")
	byn, err := Convert(rub, "BYN", 410, 100*100)
	if err != nil {
		t.Fatal(err)
	}
	if byn.Minor() != 20500 {
		t.Errorf("Convert = %d (%s), want 20500", byn.Minor(), byn.Decimal())
	}
}

func TestConvertNegative(t *testing.T) {
	usd := MustNew(-4250, "USD")
	byn, err := Convert(usd, "BYN", 325, 100)
	if err != nil {
		t.Fatal(err)
	}
	if byn.Minor() != -13813 {
		t.Errorf("Convert negative = %d, want -13813", byn.Minor())
	}
}

func TestConvertBadRate(t *testing.T) {
	usd := MustNew(100, "USD")
	if _, err := Convert(usd, "BYN", 100, 0); err == nil {
		t.Error("den=0 must error")
	}
	if _, err := Convert(usd, "", 100, 100); err == nil {
		t.Error("empty target must error")
	}
}
