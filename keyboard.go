package izpet

import (
	"sync"
	"unicode"
)

/*
The PET keyboard is a matrix of 10 rows and 8 columns. The editor ROM selects
a row with the low nibble of PIA1 port A, through a 74145 decoder, and reads
the columns on PIA1 port B, 0 for a pressed key.

There are two matrices, read from the decode tables of the editor ROMs: the
one of the 2001 chiclet and of the graphics keyboards, and the one of the
business keyboards. Only the US layouts are supported.

The keys can be pressed by position with PutKey, or by the character they
type with PutChar. A character is typed by pressing the key, and shift if
needed, for a few frames.
*/

// Key identifies a key of the PET. The keys that type a printable character
// without shift are identified by that character, 'A' or '!'.
type Key uint16

// The keys that are not identified by a character
const (
	KeyNone       Key = 0
	KeyStop       Key = 0x03
	KeyTab        Key = 0x09 // Business only
	KeyReturn     Key = 0x0d
	KeyRepeat     Key = 0x10 // Business only
	KeyCrsrDown   Key = 0x11
	KeyRvs        Key = 0x12
	KeyHome       Key = 0x13
	KeyDel        Key = 0x14
	KeyEsc        Key = 0x1b // Business only
	KeyCrsrRight  Key = 0x1d
	KeySpace      Key = 0x20
	KeyUpArrow    Key = 0x5e
	KeyLeftArrow  Key = 0x5f
	KeyShiftLeft  Key = 0x100
	KeyShiftRight Key = 0x101

	// The numeric pad of the business keyboard. The digits of the graphics
	// keyboard are on its numeric pad, but they are the only ones and are
	// identified by the characters.
	KeyKeypad0   Key = 0x130
	KeyKeypadDot Key = 0x12e
)

// KeyKeypad returns the key of a digit in the numeric pad of the business
// keyboard
func KeyKeypad(digit int) Key {
	return KeyKeypad0 + Key(digit)
}

const (
	keyboardRows        = 10
	keyboardCols        = 8
	keyboardPressFrames = 2 // Frames a key is held to type a character
	keyboardGapFrames   = 2 // Frames between characters typed
)

type keyMatrix [keyboardRows][keyboardCols]Key

// The rows start with the column on bit 0
func graphicsMatrix() *keyMatrix {
	return &keyMatrix{
		{'!', '#', '%', '&', '(', KeyLeftArrow, KeyHome, KeyCrsrRight},
		{'"', '$', '\'', '\\', ')', KeyNone, KeyCrsrDown, KeyDel},
		{'Q', 'E', 'T', 'U', 'O', KeyUpArrow, '7', '9'},
		{'W', 'R', 'Y', 'I', 'P', KeyNone, '8', '/'},
		{'A', 'D', 'G', 'J', 'L', KeyNone, '4', '6'},
		{'S', 'F', 'H', 'K', ':', KeyNone, '5', '*'},
		{'Z', 'C', 'B', 'M', ';', KeyReturn, '1', '3'},
		{'X', 'V', 'N', ',', '?', KeyNone, '2', '+'},
		{KeyShiftLeft, '@', ']', KeyNone, '>', KeyShiftRight, '0', '-'},
		{KeyRvs, '[', KeySpace, '<', KeyStop, KeyNone, '.', '='},
	}
}

func businessMatrix() *keyMatrix {
	return &keyMatrix{
		{'2', '5', '8', '-', KeyKeypad(8), KeyCrsrRight, KeyNone, KeyNone},
		{'1', '4', '7', '0', KeyKeypad(7), KeyUpArrow, KeyNone, KeyKeypad(9)},
		{KeyEsc, 'S', 'F', 'H', ']', 'K', ';', KeyKeypad(5)},
		{'A', 'D', 'G', 'J', KeyReturn, 'L', '@', KeyKeypad(6)},
		{KeyTab, 'W', 'R', 'Y', '\\', 'I', 'P', KeyDel},
		{'Q', 'E', 'T', 'U', KeyCrsrDown, 'O', '[', KeyKeypad(4)},
		{KeyShiftLeft, 'C', 'B', '.', KeyKeypadDot, KeyNone, KeyShiftRight, KeyKeypad(3)},
		{'Z', 'V', 'N', ',', KeyKeypad0, KeyNone, KeyRepeat, KeyKeypad(2)},
		{KeyRvs, 'X', KeySpace, 'M', KeyHome, KeyNone, '/', KeyKeypad(1)},
		{KeyLeftArrow, '3', '6', '9', KeyStop, ':', KeyNone, KeyNone},
	}
}

// businessShifted are the characters typed with shift on the business
// keyboard and the key that types them, as the editor ROM computes them
func businessShifted() map[rune]Key {
	return map[rune]Key{
		'!': '1', '"': '2', '#': '3', '$': '4', '%': '5',
		'&': '6', '\'': '7', '(': '8', ')': '9',
		'*': ':', '+': ';', '<': ',', '=': '-', '>': '.', '?': '/',
	}
}

type keyPosition struct {
	row int
	col int
}

type keyStroke struct {
	key   Key
	shift bool
}

type keyboard struct {
	layout    keyboardLayout
	positions map[Key]keyPosition
	shifted   map[rune]Key

	mutex   sync.Mutex
	pressed map[Key]bool
	rows    [keyboardRows]uint8 // The columns of each row, 0 for pressed

	// The characters waiting to be typed
	queue       []keyStroke
	typing      *keyStroke
	typingCount int
}

func newKeyboard(layout keyboardLayout) *keyboard {
	var k keyboard
	k.layout = layout
	k.pressed = make(map[Key]bool)
	k.positions = make(map[Key]keyPosition)

	matrix := graphicsMatrix()
	k.shifted = make(map[rune]Key)
	if layout == keyboardBusiness {
		matrix = businessMatrix()
		k.shifted = businessShifted()
	}
	for row := range matrix {
		for col, key := range matrix[row] {
			if key != KeyNone {
				k.positions[key] = keyPosition{row, col}
			}
		}
	}
	k.updateRows()
	return &k
}

// hasKey returns true if the keyboard has the key
func (k *keyboard) hasKey(key Key) bool {
	_, ok := k.positions[key]
	return ok
}

// putKey presses or releases a key
func (k *keyboard) putKey(key Key, pressed bool) {
	k.mutex.Lock()
	defer k.mutex.Unlock()
	if pressed {
		k.pressed[key] = true
	} else {
		delete(k.pressed, key)
	}
	k.updateRows()
}

// releaseAll releases all the keys pressed with putKey
func (k *keyboard) releaseAll() {
	k.mutex.Lock()
	defer k.mutex.Unlock()
	k.pressed = make(map[Key]bool)
	k.updateRows()
}

func (k *keyboard) updateRows() {
	for i := range k.rows {
		k.rows[i] = 0xff
	}
	press := func(key Key) {
		if p, ok := k.positions[key]; ok {
			k.rows[p.row] &^= 1 << p.col
		}
	}
	for key := range k.pressed {
		press(key)
	}
	if k.typing != nil && k.typingCount < keyboardPressFrames {
		press(k.typing.key)
		if k.typing.shift {
			press(KeyShiftLeft)
		}
	}
}

// readRow returns the columns of a row, rows 10 to 15 select nothing
func (k *keyboard) readRow(row uint8) uint8 {
	if row >= keyboardRows {
		return 0xff
	}
	k.mutex.Lock()
	defer k.mutex.Unlock()
	return k.rows[row]
}

// strokeForChar returns the key and shift that type a character. Letters
// are typed without shift when lowercase and with shift when uppercase, as
// the PET does in the lowercase mode.
func (k *keyboard) strokeForChar(ch rune) (keyStroke, bool) {
	switch {
	case ch == '\n' || ch == '\r':
		return keyStroke{KeyReturn, false}, true
	case ch >= 'a' && ch <= 'z':
		return keyStroke{Key(unicode.ToUpper(ch)), false}, true
	case ch >= 'A' && ch <= 'Z':
		return keyStroke{Key(ch), true}, true
	case ch == '^' || ch == '↑':
		return keyStroke{KeyUpArrow, false}, true
	case ch == '_' || ch == '←':
		return keyStroke{KeyLeftArrow, false}, true
	}

	if ch < 0x100 && k.hasKey(Key(ch)) {
		return keyStroke{Key(ch), false}, true
	}
	if key, ok := k.shifted[ch]; ok {
		return keyStroke{key, true}, true
	}
	return keyStroke{}, false
}

// putChar queues a character to be typed. It returns false if there is no
// key for it.
func (k *keyboard) putChar(ch rune) bool {
	stroke, ok := k.strokeForChar(ch)
	if !ok {
		return false
	}
	k.mutex.Lock()
	defer k.mutex.Unlock()
	k.queue = append(k.queue, stroke)
	return true
}

// putText queues the characters of a text to be typed. The uppercase
// letters are typed without shift, as needed for BASIC in the graphics mode.
func (k *keyboard) putText(text string) {
	for _, ch := range text {
		k.putChar(unicode.ToLower(ch))
	}
}

// isTyping returns true while there are characters waiting to be typed
func (k *keyboard) isTyping() bool {
	k.mutex.Lock()
	defer k.mutex.Unlock()
	return k.typing != nil || len(k.queue) > 0
}

// frame advances the typing of the queued characters
func (k *keyboard) frame() {
	k.mutex.Lock()
	defer k.mutex.Unlock()

	if k.typing != nil {
		k.typingCount++
		if k.typingCount >= keyboardPressFrames+keyboardGapFrames {
			k.typing = nil
		}
	}
	if k.typing == nil && len(k.queue) > 0 {
		k.typing = &k.queue[0]
		k.queue = k.queue[1:]
		k.typingCount = 0
	}
	k.updateRows()
}

// isKeyDown returns true when the key is down on the matrix, pressed with
// putKey or typing a character
func (k *keyboard) isKeyDown(key Key) bool {
	p, ok := k.positions[key]
	if !ok {
		return false
	}
	k.mutex.Lock()
	defer k.mutex.Unlock()
	return k.rows[p.row]&(1<<p.col) == 0
}

// keys returns the keys of the keyboard
func (k *keyboard) keys() []Key {
	keys := make([]Key, 0, len(k.positions))
	for key := range k.positions {
		keys = append(keys, key)
	}
	return keys
}
