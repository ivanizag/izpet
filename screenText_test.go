package izpet

import (
	"testing"
)

func TestScreenCodeToRune(t *testing.T) {
	if n := len(graphicsRunes()); n != 64 {
		t.Fatalf("%v graphics, expected 64", n)
	}
	cases := []struct {
		code      uint8
		lowerCase bool
		swapCase  bool
		r         rune
	}{
		{0x01, false, false, 'A'},
		{0x01, true, false, 'a'},
		{0x41, false, false, '♠'},
		{0x41, true, false, 'A'},
		{0x01, true, true, 'A'}, // The character ROM of BASIC 1
		{0x41, true, true, 'a'},
		{0x5e, false, false, 'π'},
		{0x7a, true, false, '✓'},
		{0x62, false, false, '▄'},
		{0xe2, false, false, '▀'}, // Reverse
		{0xa0, false, false, '█'}, // Reverse space
		{0x81, false, false, 'A'}, // Reverse letter, as it is
	}
	for _, c := range cases {
		if r := screenCodeToRune(c.code, c.lowerCase, c.swapCase); r != c.r {
			t.Errorf("$%02x lowercase %v swap %v: got %q, want %q", c.code, c.lowerCase, c.swapCase, r, c.r)
		}
	}
}
