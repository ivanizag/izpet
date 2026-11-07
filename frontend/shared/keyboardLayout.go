package shared

import (
	"image"
	"image/color"
	"math"

	"github.com/ivanizag/izpet"
)

/*
The keyboards of the PET models, drawn below the screen to press the keys
with the mouse.

The positions are in key units, the distance between two keys on a row, as
keyboards are made. The rows, widths and colors follow photos of the real
keyboards, the legends are in Michroma, close to the Microgramma Commodore
used, and the graphics symbols are taken from the character ROM.
*/

type keyboardLayoutKind int

const (
	layoutChiclet keyboardLayoutKind = iota
	layoutGraphics
	layoutBusiness
)

// keyStyle are the colors of a key
type keyStyle struct {
	cap    color.RGBA // The body of the key, seen around the top
	face   color.RGBA // The top of the key
	legend color.RGBA
	symbol color.RGBA // The graphics symbol
}

// DiagramKey is a key on the diagram
type DiagramKey struct {
	Key        izpet.Key
	x, y, w, h float64 // In key units
	style      *keyStyle
	top        string // Legend on the top half, the shifted character
	bottom     string // Legend on the bottom half, or the only legend
	glyph      bool   // Show the graphics symbol of the key
	vertical   bool   // The legend is written top to bottom
	Lock       bool   // A locking key, SHIFT LOCK
}

// panel is a block of keys drawn on a recess of the case
type panel struct {
	x, y, w, h float64
}

// KeyboardDiagram is the drawing of the keyboard of a model
type KeyboardDiagram struct {
	kind   keyboardLayoutKind
	width  float64 // In key units
	height float64
	keys   []DiagramKey
	panels []panel

	caseColor  color.RGBA
	panelColor color.RGBA
	capSize    float64 // The part of the pitch the key takes
	chiclet    bool    // Flat keys with a face inside a frame

	fonts *keyboardFonts
	glyph func(uint8) [8]uint8 // The character ROM
}

func rgb(c uint32) color.RGBA {
	return color.RGBA{uint8(c >> 16), uint8(c >> 8), uint8(c), 0xff}
}

// legendFor returns the legend of a character key, Commodore wrote the zero
// with a slash
func legendFor(key izpet.Key) string {
	switch key {
	case '0':
		return "Ø"
	case izpet.KeyUpArrow:
		return "↑"
	case izpet.KeyLeftArrow:
		return "←"
	}
	return string(rune(key))
}

// specs to build the rows

type spec struct {
	key      izpet.Key
	w, h     float64
	style    *keyStyle
	top      string
	bottom   string
	glyph    bool
	vertical bool
	lock     bool
}

type rowBuilder struct {
	d     *KeyboardDiagram
	x, y  float64
	style *keyStyle // For the character keys
	glyph bool      // The character keys show their graphics symbol
}

func (d *KeyboardDiagram) row(x, y float64, style *keyStyle, glyph bool) *rowBuilder {
	return &rowBuilder{d, x, y, style, glyph}
}

// chars adds keys of one unit for the characters
func (r *rowBuilder) chars(chars string) *rowBuilder {
	for _, c := range chars {
		r.char(izpet.Key(c))
	}
	return r
}

// char adds a character key
func (r *rowBuilder) char(key izpet.Key) *rowBuilder {
	return r.add(spec{key: key, w: 1, bottom: legendFor(key), glyph: r.glyph})
}

// shifted adds a business key with the shifted character on top
func (r *rowBuilder) shifted(key izpet.Key, top string) *rowBuilder {
	return r.add(spec{key: key, w: 1, top: top, bottom: legendFor(key)})
}

// special adds a key with a text legend
func (r *rowBuilder) special(key izpet.Key, w float64, style *keyStyle, legend string) *rowBuilder {
	return r.add(spec{key: key, w: w, style: style, bottom: legend})
}

func (r *rowBuilder) gap(w float64) *rowBuilder {
	r.x += w
	return r
}

func (r *rowBuilder) add(s spec) *rowBuilder {
	if s.h == 0 {
		s.h = 1
	}
	if s.style == nil {
		s.style = r.style
	}
	r.d.keys = append(r.d.keys, DiagramKey{
		Key: s.key, x: r.x, y: r.y, w: s.w, h: s.h, style: s.style,
		top: s.top, bottom: s.bottom, glyph: s.glyph, vertical: s.vertical, Lock: s.lock,
	})
	r.x += s.w
	return r
}

// NewKeyboardDiagram returns the diagram of the keyboard of the model
func NewKeyboardDiagram(p *izpet.PET) (*KeyboardDiagram, error) {
	d := layoutFor(p)
	var err error
	d.fonts, err = newKeyboardFonts()
	if err != nil {
		return nil, err
	}
	d.glyph = p.GetGlyph
	return d, nil
}

func layoutFor(p *izpet.PET) *KeyboardDiagram {
	switch {
	case p.IsBusinessKeyboard():
		return businessDiagram()
	case p.IsChicletKeyboard():
		return chicletDiagram()
	default:
		return graphicsDiagram()
	}
}

/*
The 2001 keyboard: flat square keys with a recessed face, in a dark frame.
The colors are the ones of the drawing by E5frog on Wikimedia Commons, "PET
Keyboard improved.svg", CC0.
*/
func chicletDiagram() *KeyboardDiagram {
	d := &KeyboardDiagram{
		kind:       layoutChiclet,
		width:      17.2,
		height:     6.0,
		caseColor:  rgb(0xe9e3c9),
		panelColor: rgb(0x4a4440),
		capSize:    0.92,
		chiclet:    true,
	}

	cap := rgb(0xf5f1d8)
	black := rgb(0x101010)
	white := rgb(0xffffff)
	blue := &keyStyle{cap, rgb(0xaabbd9), black, black}
	lavender := &keyStyle{cap, rgb(0xd6cedb), black, black}
	tan := &keyStyle{cap, rgb(0xd6ccb3), black, black}
	red := &keyStyle{cap, rgb(0xd41e22), white, white}
	dark := &keyStyle{cap, rgb(0x2354ba), white, white}

	x0, y0 := 0.5, 0.5
	d.panels = []panel{{x0 - 0.2, y0 - 0.2, 11.4, 5.4}, {x0 + 11.8, y0 - 0.2, 4.4, 5.4}}

	d.row(x0, y0, blue, true).chars("!\"#$%'&\\()").char(izpet.KeyLeftArrow)
	d.row(x0, y0+1, lavender, true).chars("QWERTYUIOP")
	d.row(x0+10, y0+1, blue, true).char(izpet.KeyUpArrow)
	d.row(x0, y0+2, lavender, true).chars("ASDFGHJKL")
	d.row(x0+9, y0+2, blue, true).char(':')
	d.row(x0+10, y0+2, red, false).add(spec{key: izpet.KeyReturn, w: 1, h: 2, bottom: "RETURN", vertical: true})
	d.row(x0, y0+3, lavender, true).chars("ZXCVBNM")
	d.row(x0+7, y0+3, blue, true).chars(",;?")
	d.row(x0, y0+4, blue, true).
		special(izpet.KeyShiftLeft, 1, red, "SHIFT").
		special(izpet.KeyRvs, 1, dark, "OFF\nRVS").
		chars("@[]").
		special(izpet.KeySpace, 2, red, "SPACE").
		chars("<>").
		special(izpet.KeyStop, 1, dark, "RUN\nSTOP").
		special(izpet.KeyShiftRight, 1, red, "SHIFT")

	x1 := x0 + 12
	d.row(x1, y0, blue, true).
		special(izpet.KeyHome, 1, red, "CLR\nHOME").
		special(izpet.KeyCrsrDown, 1, dark, "↑\nCRSR\n↓").
		special(izpet.KeyCrsrRight, 1, dark, "←\nCRSR\n→").
		special(izpet.KeyDel, 1, red, "INST\nDEL")
	d.row(x1, y0+1, tan, true).chars("789").row2(blue, "/")
	d.row(x1, y0+2, tan, true).chars("456").row2(blue, "*")
	d.row(x1, y0+3, tan, true).chars("123").row2(blue, "+")
	d.row(x1, y0+4, tan, true).chars("0.").row2(blue, "-=")
	return d
}

// row2 continues the row with another style
func (r *rowBuilder) row2(style *keyStyle, chars string) *rowBuilder {
	saved := r.style
	r.style = style
	r.chars(chars)
	r.style = saved
	return r
}

// The graphics keyboard of the 2001-N, 3000 and 4000: dark sculpted keys, the
// characters in gold and the graphics symbols in white
func graphicsDiagram() *KeyboardDiagram {
	d := &KeyboardDiagram{
		kind:       layoutGraphics,
		width:      20.3,
		height:     6.0,
		caseColor:  rgb(0xd8cba0),
		panelColor: rgb(0x2a2320),
		capSize:    0.94,
	}

	gold := rgb(0xd9b45a)
	white := rgb(0xe6e6dc)
	grey := rgb(0xcfcfc6)
	key := &keyStyle{rgb(0x1a1412), rgb(0x302825), gold, white}
	special := &keyStyle{rgb(0x1a1412), rgb(0x302825), grey, white}
	pad := &keyStyle{rgb(0x30353e), rgb(0x4b5260), white, white}

	x0, y0 := 0.4, 0.5
	d.panels = []panel{{x0 - 0.15, y0 - 0.15, 14.8, 5.3}, {x0 + 15.35, y0 - 0.15, 4.3, 5.3}}

	d.row(x0+0.25, y0, key, true).chars("@!\"#$%'&\\()").char(izpet.KeyLeftArrow).chars("[]")
	d.row(x0, y0+1, key, true).special(izpet.KeyRvs, 1.5, special, "OFF\nRVS").
		chars("QWERTYUIOP").char(izpet.KeyUpArrow).chars("<>")
	d.row(x0, y0+2, key, true).
		add(spec{key: izpet.KeyShiftLeft, w: 1.75, style: special, bottom: "SHIFT\nLOCK", lock: true}).
		chars("ASDFGHJKL:").
		special(izpet.KeyStop, 1, special, "RUN\nSTOP").
		special(izpet.KeyReturn, 1.75, special, "RETURN")
	d.row(x0, y0+3, key, true).special(izpet.KeyShiftLeft, 2.25, special, "SHIFT").
		chars("ZXCVBNM,;?").special(izpet.KeyShiftRight, 2.25, special, "SHIFT")
	d.row(x0+2.5, y0+4, key, false).special(izpet.KeySpace, 9.6, special, "")

	x1 := x0 + 15.5
	d.row(x1, y0, pad, false).
		special(izpet.KeyHome, 1, special, "CLR\nHOME").
		special(izpet.KeyCrsrDown, 1, special, "↑\nCRSR\n↓").
		special(izpet.KeyCrsrRight, 1, special, "←\nCRSR\n→").
		special(izpet.KeyDel, 1, special, "INST\nDEL")
	d.row(x1, y0+1, pad, true).chars("789/")
	d.row(x1, y0+2, pad, true).chars("456*")
	d.row(x1, y0+3, pad, true).chars("123+")
	d.row(x1, y0+4, pad, true).chars("0.-=")
	return d
}

// The business keyboard of the 8000 and the B models: dark keys with white
// legends, the shifted character over the unshifted one
func businessDiagram() *KeyboardDiagram {
	d := &KeyboardDiagram{
		kind:       layoutBusiness,
		width:      20.6,
		height:     6.0,
		caseColor:  rgb(0xd4d5d2),
		panelColor: rgb(0xb8b9b6),
		capSize:    0.94,
	}

	key := &keyStyle{rgb(0x1e1f20), rgb(0x3a3b3d), rgb(0xececec), rgb(0xececec)}

	x0, y0 := 0.4, 0.5
	d.panels = []panel{{x0 - 0.15, y0 - 0.15, 16.3, 5.3}, {x0 + 16.45, y0 - 0.15, 3.3, 4.3}}

	r := d.row(x0+0.4, y0, key, false).char(izpet.KeyLeftArrow)
	for i, top := range []string{"!", "\"", "#", "$", "%", "&", "'", "(", ")"} {
		r.shifted(izpet.Key('1'+i), top)
	}
	r.char('0').shifted(':', "*").shifted('-', "=").char(izpet.KeyUpArrow).
		special(izpet.KeyCrsrRight, 1, key, "←  →\nCRSR").
		special(izpet.KeyStop, 1, key, "RUN\nSTOP")
	d.row(x0, y0+1, key, false).special(izpet.KeyTab, 1.75, key, "TAB").
		chars("QWERTYUIOP[\\").
		special(izpet.KeyCrsrDown, 1, key, "↑\nCRSR\n↓").
		special(izpet.KeyDel, 1, key, "INST\nDEL")
	d.row(x0, y0+2, key, false).special(izpet.KeyEsc, 1, key, "ESC").
		add(spec{key: izpet.KeyShiftLeft, w: 1, bottom: "SHIFT\nLOCK", lock: true}).
		chars("ASDFGHJKL").shifted(';', "+").chars("@]").
		special(izpet.KeyReturn, 1.8, key, "RETURN")
	d.row(x0, y0+3, key, false).special(izpet.KeyRvs, 1, key, "OFF\nRVS").
		special(izpet.KeyShiftLeft, 1.5, key, "SHIFT").
		chars("ZXCVBNM").shifted(',', "<").shifted('.', ">").shifted('/', "?").
		special(izpet.KeyShiftRight, 1.5, key, "SHIFT").
		special(izpet.KeyRepeat, 1, key, "REPEAT").
		special(izpet.KeyHome, 1, key, "CLR\nHOME")
	d.row(x0+2.6, y0+4, key, false).special(izpet.KeySpace, 9.2, key, "")

	x1 := x0 + 16.6
	d.row(x1, y0, key, false).add(pad(7)).add(pad(8)).add(pad(9))
	d.row(x1, y0+1, key, false).add(pad(4)).add(pad(5)).add(pad(6))
	d.row(x1, y0+2, key, false).add(pad(1)).add(pad(2)).add(pad(3))
	d.row(x1, y0+3, key, false).add(spec{key: izpet.KeyKeypad0, w: 2, bottom: "Ø"}).
		add(spec{key: izpet.KeyKeypadDot, w: 1, bottom: "."})
	return d
}

func pad(digit int) spec {
	return spec{key: izpet.KeyKeypad(digit), w: 1, bottom: string(rune('0' + digit))}
}

// Aspect returns the height of the diagram for a width of one
func (d *KeyboardDiagram) Aspect() float64 {
	return d.height / d.width
}

// Keys returns the keys on the diagram
func (d *KeyboardDiagram) Keys() []DiagramKey {
	return d.keys
}

// KeyAt returns the key at a position in pixels of the diagram drawn at a
// width
func (d *KeyboardDiagram) KeyAt(x, y float64, width int) (DiagramKey, bool) {
	unit := float64(width) / d.width
	x /= unit
	y /= unit
	for _, k := range d.keys {
		if x >= k.x && x < k.x+k.w && y >= k.y && y < k.y+k.h {
			return k, true
		}
	}
	return DiagramKey{}, false
}

// KeyArea returns the area in pixels of a key on the diagram drawn at a
// width, with the gap around it up to the next key
func (d *KeyboardDiagram) KeyArea(k *DiagramKey, width int) image.Rectangle {
	unit := float64(width) / d.width
	body, _ := d.keyRects(k, unit)
	gap := d.gap(unit)
	return image.Rect(int(math.Round(body.x-gap)), int(math.Round(body.y-gap)),
		int(math.Round(body.x+body.w+gap)), int(math.Round(body.y+body.h+gap)))
}
