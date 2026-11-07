package izpet

import (
	"testing"
)

func TestRuneToScreenCode(t *testing.T) {
	cases := []struct {
		ch       rune
		swapCase bool
		code     uint8
	}{
		{'@', false, 0x00},
		{'a', false, 0x01},
		{'A', false, 0x41},
		{'a', true, 0x41},
		{'Z', true, 0x1a},
		{'[', false, 0x1b},
		{'1', false, 0x31},
		{'~', false, '?'},
	}
	for _, c := range cases {
		if got := runeToScreenCode(c.ch, c.swapCase); got != c.code {
			t.Errorf("%q swap %v: got $%02x, want $%02x", c.ch, c.swapCase, got, c.code)
		}
	}
}

func TestMessageImage(t *testing.T) {
	for _, model := range []string{"2001", "4032", "8032"} {
		p := newTestPET(t, model)
		empty := p.GetMessageImage("")
		img := p.GetMessageImage("Help")
		if img.Bounds() != p.GetImage().Bounds() {
			t.Errorf("%v: the message is %v, the screen %v", model, img.Bounds(), p.GetImage().Bounds())
		}
		lit := 0
		for i := 1; i < len(img.Pix); i += 4 {
			if img.Pix[i] != empty.Pix[i] {
				lit++
			}
		}
		if lit < 20 {
			t.Errorf("%v: the message has %v pixels lit", model, lit)
		}
	}
}
