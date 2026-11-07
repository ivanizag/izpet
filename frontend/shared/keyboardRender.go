package shared

import (
	_ "embed"
	"image"
	"image/color"
	"image/draw"
	"math"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"

	"github.com/ivanizag/izpet"
)

/*
Drawing of the keyboard diagrams, on the CPU, at the size they are shown. It
runs again only when the size of the window changes.
*/

//go:embed fonts/Michroma-Regular.ttf
var michromaTTF []byte

type keyboardFonts struct {
	michroma *opentype.Font
	fallback *opentype.Font // For the arrows, missing in Michroma
	faces    map[fontKey]font.Face
}

type fontKey struct {
	fallback bool
	size     float64
}

func newKeyboardFonts() (*keyboardFonts, error) {
	var f keyboardFonts
	var err error
	f.michroma, err = opentype.Parse(michromaTTF)
	if err != nil {
		return nil, err
	}
	f.fallback, err = opentype.Parse(gobold.TTF)
	if err != nil {
		return nil, err
	}
	f.faces = make(map[fontKey]font.Face)
	return &f, nil
}

func (f *keyboardFonts) face(fallback bool, size float64) font.Face {
	key := fontKey{fallback, math.Round(size*4) / 4}
	if face, ok := f.faces[key]; ok {
		return face
	}
	source := f.michroma
	if fallback {
		source = f.fallback
	}
	face, err := opentype.NewFace(source, &opentype.FaceOptions{
		Size: key.size, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		panic(err)
	}
	f.faces[key] = face
	return face
}

// faceFor returns the face with the glyph for a rune, Michroma if it has it
func (f *keyboardFonts) faceFor(r rune, size float64) font.Face {
	face := f.face(false, size)
	if _, ok := face.GlyphAdvance(r); ok && r < 0x2000 {
		return face
	}
	return f.face(true, size)
}

// runeScale makes the punctuation bigger, it is small in Michroma and was
// bold and big on the Commodore keys
func runeScale(r rune) float64 {
	if strings.ContainsRune("\"',.;:-", r) {
		return 1.7
	}
	return 1
}

func (f *keyboardFonts) measure(text string, size float64) float64 {
	width := fixed.Int26_6(0)
	for _, r := range text {
		advance, _ := f.faceFor(r, size*runeScale(r)).GlyphAdvance(r)
		width += advance
	}
	return float64(width) / 64
}

// drawText draws a line of text centered on x, with the baseline at y. Bold
// is made drawing it twice, Michroma has no bold.
func (f *keyboardFonts) drawText(dst *image.RGBA, text string, x, y, size float64, c color.RGBA, bold bool) {
	start := x - f.measure(text, size)/2
	passes := []float64{0}
	if bold {
		passes = append(passes, math.Max(1, size/14))
	}
	src := image.NewUniform(c)
	for _, offset := range passes {
		d := font.Drawer{Dst: dst, Src: src}
		d.Dot = fixed.Point26_6{X: fixed.Int26_6((start + offset) * 64), Y: fixed.Int26_6(y * 64)}
		for _, r := range text {
			d.Face = f.faceFor(r, size*runeScale(r))
			d.DrawString(string(r))
		}
	}
}

// rect is a rectangle in pixels, with float coordinates
type rect struct {
	x, y, w, h float64
}

func (r rect) inset(left, top, right, bottom float64) rect {
	return rect{r.x + left, r.y + top, r.w - left - right, r.h - top - bottom}
}

// fillRoundRect draws a rectangle with rounded corners, antialiased
func fillRoundRect(dst *image.RGBA, r rect, radius float64, c color.RGBA) {
	bx, by := int(math.Floor(r.x)), int(math.Floor(r.y))
	bw, bh := int(math.Ceil(r.x+r.w))-bx, int(math.Ceil(r.y+r.h))-by
	if bw <= 0 || bh <= 0 {
		return
	}
	radius = math.Min(radius, math.Min(r.w, r.h)/2)
	x0, y0 := float32(r.x-float64(bx)), float32(r.y-float64(by))
	x1, y1 := x0+float32(r.w), y0+float32(r.h)
	rad := float32(radius)

	z := vector.NewRasterizer(bw, bh)
	z.MoveTo(x0+rad, y0)
	z.LineTo(x1-rad, y0)
	z.QuadTo(x1, y0, x1, y0+rad)
	z.LineTo(x1, y1-rad)
	z.QuadTo(x1, y1, x1-rad, y1)
	z.LineTo(x0+rad, y1)
	z.QuadTo(x0, y1, x0, y1-rad)
	z.LineTo(x0, y0+rad)
	z.QuadTo(x0, y0, x0+rad, y0)
	z.ClosePath()
	z.Draw(dst, image.Rect(bx, by, bx+bw, by+bh), image.NewUniform(c), image.Point{})
}

// drawGlyph draws a character of the character ROM, size is its width
func drawGlyph(dst *image.RGBA, glyph [8]uint8, x, y, size float64, c color.RGBA) {
	pixel := size / 8
	for row := 0; row < 8; row++ {
		for col := 0; col < 8; col++ {
			if glyph[row]&(0x80>>col) == 0 {
				continue
			}
			r := image.Rect(
				int(math.Round(x+float64(col)*pixel)), int(math.Round(y+float64(row)*pixel)),
				int(math.Round(x+float64(col+1)*pixel)), int(math.Round(y+float64(row+1)*pixel)))
			draw.Draw(dst, r, image.NewUniform(c), image.Point{}, draw.Src)
		}
	}
}

// symbolScreenCode returns the screen code of the graphics symbol typed with
// shift on a character key
func symbolScreenCode(key izpet.Key) (uint8, bool) {
	switch {
	case key >= 0x20 && key < 0x40 && key != ' ':
		return uint8(key) + 0x40, true
	case key >= 0x40 && key < 0x60:
		return uint8(key), true
	}
	return 0, false
}

// keyRects returns where the body and the top of a key are, in pixels
func (d *KeyboardDiagram) keyRects(k *DiagramKey, unit float64) (body rect, face rect) {
	margin := (1 - d.capSize) / 2 * unit
	body = rect{k.x * unit, k.y * unit, k.w * unit, k.h * unit}.inset(margin, margin, margin, margin)
	if d.chiclet {
		face = body.inset(0.1*unit, 0.1*unit, 0.1*unit, 0.1*unit)
	} else {
		face = body.inset(0.09*unit, 0.04*unit, 0.09*unit, 0.15*unit)
	}
	return body, face
}

// gap returns half the space between two keys, in pixels
func (d *KeyboardDiagram) gap(unit float64) float64 {
	return (1 - d.capSize) / 2 * unit
}

// Size returns the size in pixels of the diagram drawn at a width
func (d *KeyboardDiagram) Size(width int) (int, int) {
	return width, int(math.Round(float64(width) * d.height / d.width))
}

// Render draws the keyboard at a width in pixels
func (d *KeyboardDiagram) Render(width int) *image.RGBA {
	return d.render(width, false)
}

/*
RenderPressed draws the keyboard with all the keys pressed: pushed down, with
a ring around them in the gap between the keys. The legends keep their colors,
to read them as well as on the keys up. A frontend shows the keys down taking
their area, see KeyArea, from this image.
*/
func (d *KeyboardDiagram) RenderPressed(width int) *image.RGBA {
	return d.render(width, true)
}

// pressedRing is the color of the ring around the keys down
var pressedRing = color.RGBA{0x3c, 0xb4, 0x4b, 0xff}

func (d *KeyboardDiagram) render(width int, pressed bool) *image.RGBA {
	fonts := d.fonts
	glyph := d.glyph
	w, h := d.Size(width)
	unit := float64(w) / d.width
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), image.NewUniform(d.caseColor), image.Point{}, draw.Src)

	for _, p := range d.panels {
		fillRoundRect(img, rect{p.x * unit, p.y * unit, p.w * unit, p.h * unit}, 0.12*unit, d.panelColor)
	}

	for i := range d.keys {
		k := &d.keys[i]
		body, face := d.keyRects(k, unit)
		if pressed {
			// The ring takes the gap around the key
			ring := d.gap(unit)
			fillRoundRect(img, body.inset(-ring, -ring, -ring, -ring), 0.1*unit+ring, pressedRing)

			// The key goes down: it looks smaller, more so at the top, and
			// the ring is seen around it
			in := 0.03 * unit
			down := 0.02 * unit
			body = body.inset(in, in+down, in, in)
			face = face.inset(in*0.5, in*0.5+down, in*0.5, in*0.5)
		}
		if d.chiclet {
			fillRoundRect(img, body.inset(-0.015*unit, -0.015*unit, -0.015*unit, -0.015*unit), 0.07*unit, rgb(0xb8b090))
			fillRoundRect(img, body, 0.06*unit, k.style.cap)
			fillRoundRect(img, face, 0.04*unit, k.style.face)
		} else {
			fillRoundRect(img, body, 0.1*unit, k.style.cap)
			fillRoundRect(img, face, 0.08*unit, k.style.face)
		}
		d.renderLegends(img, k, face, unit, fonts, glyph)
	}
	return img
}

func (d *KeyboardDiagram) renderLegends(img *image.RGBA, k *DiagramKey, face rect, unit float64,
	fonts *keyboardFonts, glyph func(uint8) [8]uint8) {
	cx := face.x + face.w/2
	style := k.style
	code, hasSymbol := symbolScreenCode(k.Key)
	hasSymbol = hasSymbol && k.glyph

	switch {
	case k.vertical:
		size := 0.17 * unit
		letters := []rune(k.bottom)
		step := face.h / float64(len(letters)+1)
		for i, r := range letters {
			fonts.drawText(img, string(r), cx, face.y+step*float64(i+1)+size*0.35, size, style.legend, true)
		}

	case strings.Contains(k.bottom, "\n") || len([]rune(k.bottom)) > 2:
		// A text legend
		lines := strings.Split(k.bottom, "\n")
		size := 0.15 * unit
		if d.chiclet {
			size = 0.135 * unit
		}
		for _, line := range lines {
			if width := fonts.measure(line, size); width > face.w*0.92 {
				size *= face.w * 0.92 / width
			}
		}
		lineHeight := size * 1.25
		y := face.y + (face.h-lineHeight*float64(len(lines)))/2 + size
		for _, line := range lines {
			fonts.drawText(img, line, cx, y, size, style.legend, d.chiclet)
			y += lineHeight
		}

	case d.kind == layoutChiclet:
		// The graphics symbol on top, the character below
		if hasSymbol {
			s := 0.24 * unit
			drawGlyph(img, glyph(code), cx-s/2, face.y+0.08*unit, s, style.symbol)
		}
		fonts.drawText(img, k.bottom, cx, face.y+face.h-0.08*unit, 0.25*unit, style.legend, true)

	case d.kind == layoutGraphics:
		// The character on top, the graphics symbol below
		fonts.drawText(img, k.bottom, cx, face.y+0.32*unit, 0.24*unit, style.legend, true)
		if hasSymbol {
			s := 0.22 * unit
			drawGlyph(img, glyph(code), cx-s/2, face.y+face.h-s-0.08*unit, s, style.symbol)
		}

	case k.top != "":
		// The shifted character over the unshifted one
		size := 0.22 * unit
		fonts.drawText(img, k.top, cx, face.y+0.3*unit, size, style.legend, true)
		fonts.drawText(img, k.bottom, cx, face.y+face.h-0.08*unit, size, style.legend, true)

	default:
		fonts.drawText(img, k.bottom, cx, face.y+face.h/2+0.1*unit, 0.26*unit, style.legend, true)
	}
}
