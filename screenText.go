package izpet

import (
	"strings"
)

/*
The text on the screen, from the screen codes in the screen RAM. The screen
codes are not PETSCII: 0 to 31 are @, the letters and [\]↑←, 32 to 63 are as
ASCII, and 64 to 127 the graphics, or the uppercase letters in the lowercase
mode. Bit 7 is reverse video.
*/

// GetScreenText returns the text on the screen, a line per row
func (p *PET) GetScreenText() string {
	columns := 40
	if p.model.video == videoCRTC80 {
		columns = 80
	}
	screen := p.mmu.screenRAM()
	lowerCase := p.io.isLowerCase()

	var sb strings.Builder
	for row := 0; row < 25; row++ {
		line := make([]rune, columns)
		for col := 0; col < columns; col++ {
			line[col] = screenCodeToRune(screen[row*columns+col], lowerCase)
		}
		sb.WriteString(strings.TrimRight(string(line), " "))
		sb.WriteString("\n")
	}
	return sb.String()
}

func screenCodeToRune(code uint8, lowerCase bool) rune {
	code &= 0x7f
	switch {
	case code == 0x00:
		return '@'
	case code <= 0x1a:
		if lowerCase {
			return 'a' + rune(code-1)
		}
		return 'A' + rune(code-1)
	case code < 0x20:
		return []rune{'[', '\\', ']', '↑', '←'}[code-0x1b]
	case code < 0x40:
		return rune(code)
	case lowerCase && code >= 0x41 && code <= 0x5a:
		return 'A' + rune(code-0x41)
	case code == 0x60:
		return ' ' // Shifted space
	default:
		return '·' // A graphics character
	}
}
