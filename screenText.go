package izpet

import (
	"strings"
)

/*
The text on the screen, from the screen codes in the screen RAM. The screen
codes are not PETSCII: 0 to 31 are @, the letters and [\]↑←, 32 to 63 are as
ASCII, and 64 to 127 the graphics, or the uppercase letters in the lowercase
mode. Bit 7 is reverse video.

The graphics are the Unicode characters that look the same, from the box
drawing and block elements blocks, and from the Symbols for Legacy Computing
block, U+1FB00, that has the ones of the PET and of the computers after it.
They were checked against the glyphs of the character ROM. The character
ROM of BASIC 1 has the letters of the lowercase set in the other case.

The characters in reverse video are the Unicode characters of the inverse
glyph, the reverse space a full block, when there is one. The others are
shown as if they were not in reverse.
*/

// GetScreenText returns the text on the screen, a line per row
func (p *PET) GetScreenText() string {
	columns := 40
	if p.model.video == videoCRTC80 {
		columns = 80
	}
	screen := p.mmu.screenRAM()
	lowerCase := p.io.isLowerCase()
	swapCase := lowerCase && p.model.basic == 1

	var sb strings.Builder
	for row := 0; row < 25; row++ {
		line := make([]rune, columns)
		for col := 0; col < columns; col++ {
			line[col] = screenCodeToRune(screen[row*columns+col], lowerCase, swapCase)
		}
		sb.WriteString(strings.TrimRight(string(line), " "))
		sb.WriteString("\n")
	}
	return sb.String()
}

// graphicsRunes are the characters of the screen codes 64 to 127 in the
// graphics set
func graphicsRunes() []rune {
	return []rune{
		// $40
		'─', '♠', '\U0001FB72', '\U0001FB78', '\U0001FB77', '\U0001FB76', '\U0001FB7A', '\U0001FB71',
		'\U0001FB74', '╮', '╰', '╯', '\U0001FB7C', '╲', '╱', '\U0001FB7D',
		// $50
		'\U0001FB7E', '●', '\U0001FB7B', '♥', '\U0001FB70', '╭', '╳', '○',
		'♣', '\U0001FB75', '♦', '┼', '\U0001FB8C', '│', 'π', '◥',
		// $60
		' ', '▌', '▄', '▔', '▁', '▏', '▒', '▕',
		'\U0001FB8F', '◤', '\U0001FB87', '├', '▗', '└', '┐', '▂',
		// $70
		'┌', '┴', '┬', '┤', '▎', '▍', '\U0001FB88', '\U0001FB82',
		'\U0001FB83', '▃', '\U0001FB7F', '▖', '▝', '┘', '▘', '▚',
	}
}

// lowerCaseRunes are the graphics of the lowercase set that are not the ones
// of the graphics set
func lowerCaseRunes() map[uint8]rune {
	return map[uint8]rune{
		0x5e: '\U0001FB95', // Checker board
		0x5f: '\U0001FB98', // Upper left to lower right fill
		0x69: '\U0001FB99', // Upper right to lower left fill
		0x7a: '✓',
	}
}

// inverseRunes are the characters of the inverse glyphs
func inverseRunes() map[rune]rune {
	return map[rune]rune{
		' ':          '█',
		'▌':          '▐',
		'▄':          '▀',
		'▔':          '▇',
		'▁':          '\U0001FB86',
		'▏':          '\U0001FB8B',
		'▕':          '▉',
		'▒':          '\U0001FB90',
		'▗':          '▛',
		'▖':          '▜',
		'▝':          '▙',
		'▘':          '▟',
		'▚':          '▞',
		'▂':          '\U0001FB85',
		'▃':          '\U0001FB84',
		'▎':          '\U0001FB8A',
		'▍':          '\U0001FB89',
		'\U0001FB87': '▊',
		'\U0001FB88': '▋',
		'\U0001FB82': '▆',
		'\U0001FB83': '▅',
	}
}

func screenCodeToRune(code uint8, lowerCase bool, swapCase bool) rune {
	r := plainScreenCodeToRune(code&0x7f, lowerCase, swapCase)
	if code&0x80 != 0 {
		if inverse, ok := inverseRunes()[r]; ok {
			return inverse
		}
	}
	return r
}

func plainScreenCodeToRune(code uint8, lowerCase bool, swapCase bool) rune {
	switch {
	case code == 0x00:
		return '@'
	case code <= 0x1a:
		if lowerCase != swapCase {
			return 'a' + rune(code-1)
		}
		return 'A' + rune(code-1)
	case code < 0x20:
		return []rune{'[', '\\', ']', '↑', '←'}[code-0x1b]
	case code < 0x40:
		return rune(code)
	case lowerCase && code >= 0x41 && code <= 0x5a:
		if swapCase {
			return 'a' + rune(code-0x41)
		}
		return 'A' + rune(code-0x41)
	}
	if lowerCase {
		if r, ok := lowerCaseRunes()[code]; ok {
			return r
		}
	}
	return graphicsRunes()[code-0x40]
}
