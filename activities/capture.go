package activities

import (
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"io"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"github.com/ivanizag/izpet"
)

/*
The frame around a screenshot: black, as the glass of the screen of the PET is
around what it shows, with rounded corners. A label, when there is one, goes
in its bottom edge in grey, the way a name is on the front of a monitor.

The screen is shown with its pixels in their shape, as the frontend does:
each scan line twice, and the pixels of the 40 columns models twice as wide as
the ones of the 80 columns models.
*/
const (
	frameWidth  = 14
	frameLabel  = 14
	frameRadius = 12
)

// labelGrey is the colour of the label in the frame
var labelGrey = color.Gray{Y: 0xb0}

// screenOf returns the screen of a machine with its pixels in their shape
func screenOf(p *izpet.PET) *image.RGBA {
	src := p.GetImage()
	sx := 2
	if p.Is80Columns() {
		sx = 1
	}
	b := src.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dx()*sx, b.Dy()*2))
	for y := 0; y < out.Bounds().Dy(); y++ {
		for x := 0; x < out.Bounds().Dx(); x++ {
			out.SetRGBA(x, y, src.RGBAAt(b.Min.X+x/sx, b.Min.Y+y/2))
		}
	}
	return out
}

/*
Frame puts a screen in the frame every picture of the activities has, with a
label under it when one is given. The corners outside the frame are
transparent.
*/
func Frame(screen image.Image, label string) *image.RGBA {
	b := screen.Bounds()
	bottom := frameWidth
	if label != "" {
		bottom += frameLabel
	}
	width, height := b.Dx()+2*frameWidth, b.Dy()+frameWidth+bottom
	out := image.NewRGBA(image.Rect(0, 0, width, height))

	// Black, but for what the rounding leaves out of each corner
	corner := func(x int, last int) int {
		return max(frameRadius-x, x-(last-frameRadius), 0)
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			dx, dy := corner(x, width-1), corner(y, height-1)
			if dx*dx+dy*dy <= frameRadius*frameRadius {
				out.Set(x, y, color.Black)
			}
		}
	}
	draw.Draw(out, image.Rect(frameWidth, frameWidth, frameWidth+b.Dx(), frameWidth+b.Dy()),
		screen, b.Min, draw.Src)

	if label != "" {
		writer := font.Drawer{
			Dst:  out,
			Src:  image.NewUniform(labelGrey),
			Face: basicfont.Face7x13,
		}
		textWidth := writer.MeasureString(label).Round()
		writer.Dot = fixed.P((width-textWidth)/2, frameWidth+b.Dy()+frameLabel-1)
		writer.DrawString(label)
	}
	return out
}

/*
A Recording is an animated screenshot, for what is worth watching move: a GIF
of the screen in its frame, a picture taken every few frames of the machine.
Each picture after the first is only the part of it that changed, and a
picture that changed nothing makes the one before it last longer, up to
LongestStill.

Delays are in hundredths of a second, as a GIF counts them. Browsers slow down
anything shorter than two, so a picture is not worth taking more often than
every two frames of the machine.
*/
type Recording struct {
	p      *izpet.PET
	label  string
	frames []*image.Paletted
	delays []int
	last   *image.Paletted
}

// LongestStill is the longest a picture stays, in hundredths of a second: a
// wait on the machine with nothing changing is shortened to that
const LongestStill = 300

// recordingPalette is what a framed screenshot is made of: the corners left
// out of the frame, black, the green of the screen, and the grey of a label
func recordingPalette() color.Palette {
	return color.Palette{color.Transparent, color.Black, color.RGBA{0x41, 0xff, 0x00, 0xff}, labelGrey}
}

// Record starts a recording of a machine, with a label in its frame when one
// is given. Nothing is taken until Capture or Run.
func Record(p *izpet.PET, label string) *Recording {
	return &Recording{p: p, label: label}
}

// Capture takes a picture of the screen, shown for a time in hundredths of a
// second
func (r *Recording) Capture(delay int) {
	whole := Frame(screenOf(r.p), r.label)
	picture := image.NewPaletted(whole.Bounds(), recordingPalette())
	for i := 0; i < len(whole.Pix); i += 4 {
		var index uint8
		switch {
		case whole.Pix[i+3] == 0:
			index = 0
		case whole.Pix[i+1] == 0xff:
			index = 2
		case whole.Pix[i] == labelGrey.Y:
			index = 3
		default:
			index = 1
		}
		picture.Pix[i/4] = index
	}

	if r.last == nil {
		r.frames, r.delays, r.last = append(r.frames, picture), append(r.delays, delay), picture
		return
	}

	// Only the part that changed, if anything did
	changed := image.Rectangle{}
	bounds := picture.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if picture.ColorIndexAt(x, y) != r.last.ColorIndexAt(x, y) {
				changed = changed.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	r.last = picture
	if changed.Empty() {
		r.delays[len(r.delays)-1] = min(r.delays[len(r.delays)-1]+delay, LongestStill)
		return
	}
	part := image.NewPaletted(changed, recordingPalette())
	draw.Draw(part, changed, picture, changed.Min, draw.Src)
	r.frames, r.delays = append(r.frames, part), append(r.delays, delay)
}

// Run runs the machine for some frames, taking a picture every few of them,
// shown for as long as they took
func (r *Recording) Run(frames int, every int) {
	for done := 0; done < frames; done += every {
		r.p.RunFrames(uint64(every))
		r.Capture(every * 100 / 60)
	}
}

// Type types a text, with pictures while it is typed
func (r *Recording) Type(text string, every int) {
	r.p.TypeText(text)
	for r.p.IsTyping() {
		r.Run(every, every)
	}
}

/*
Encode writes the recording as a GIF that plays in a loop, the last picture
held for longer, by hold hundredths of a second, so that what it ends on is
seen before it starts again
*/
func (r *Recording) Encode(w io.Writer, hold int) error {
	if len(r.delays) == 0 {
		return errors.New("the recording has nothing in it")
	}
	delays := append([]int(nil), r.delays...)
	delays[len(delays)-1] += hold
	return gif.EncodeAll(w, &gif.GIF{Image: r.frames, Delay: delays})
}
