package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/ivanizag/izpet"
	"github.com/ivanizag/izpet/frontend/shared"
)

/*
The diagram of the keyboard below the screen, and the keys pressed on it with
the mouse.

A key is held while the button is. SHIFT stays down after a click, to press
another key with it, and comes up with that key. SHIFT LOCK goes down with a
click and up with the next one.

The keys down on the matrix are shown pressed, whether pressed with the
mouse, the keyboard of the host or typing. A character typed holds its key
for two frames only, enough for the editor ROM and too little to be seen, so
a key is shown pressed for keyShownTicks at least.

The diagram is drawn at the size it is shown, again when the size changes.
*/
type ebitenKeyboardDiagram struct {
	p       *izpet.PET
	diagram *shared.KeyboardDiagram
	image   *ebiten.Image
	pressed *ebiten.Image // All the keys down

	clicked izpet.Key // The key held with the mouse, KeyNone if none
	sticky  map[izpet.Key]bool
	locked  map[izpet.Key]bool

	ticks      uint64
	shownUntil map[izpet.Key]uint64 // Tick to show the key pressed until
}

// keyShownTicks is the shortest time a key is shown pressed, a sixth of a
// second at the 60 ticks per second of ebiten
const keyShownTicks = 10

func newEbitenKeyboardDiagram(p *izpet.PET) (*ebitenKeyboardDiagram, error) {
	var kd ebitenKeyboardDiagram
	kd.p = p
	kd.sticky = make(map[izpet.Key]bool)
	kd.locked = make(map[izpet.Key]bool)
	kd.shownUntil = make(map[izpet.Key]uint64)

	var err error
	kd.diagram, err = shared.NewKeyboardDiagram(p)
	if err != nil {
		return nil, err
	}
	return &kd, nil
}

// aspect returns the height of the diagram for a width of one
func (kd *ebitenKeyboardDiagram) aspect() float64 {
	return kd.diagram.Aspect()
}

// update handles the mouse. The position is in pixels of the diagram drawn
// at the width.
func (kd *ebitenKeyboardDiagram) update(x, y float64, width int) {
	kd.ticks++
	for _, k := range kd.diagram.Keys() {
		if kd.p.IsKeyDown(k.Key) {
			kd.shownUntil[k.Key] = kd.ticks + keyShownTicks
		}
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if key, ok := kd.diagram.KeyAt(x, y, width); ok {
			kd.click(key)
		}
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		kd.unclick()
	}
}

func isShift(key izpet.Key) bool {
	return key == izpet.KeyShiftLeft || key == izpet.KeyShiftRight
}

func (kd *ebitenKeyboardDiagram) click(key shared.DiagramKey) {
	switch {
	case key.Lock:
		kd.locked[key.Key] = !kd.locked[key.Key]
		kd.p.PutKey(key.Key, kd.locked[key.Key])
	case isShift(key.Key):
		kd.sticky[key.Key] = !kd.sticky[key.Key]
		kd.p.PutKey(key.Key, kd.sticky[key.Key] || kd.locked[key.Key])
	default:
		kd.clicked = key.Key
		kd.p.PutKey(key.Key, true)
	}
}

func (kd *ebitenKeyboardDiagram) unclick() {
	if kd.clicked == izpet.KeyNone {
		return
	}
	kd.p.PutKey(kd.clicked, false)
	kd.clicked = izpet.KeyNone

	// The shifts clicked before go up with the key
	for key := range kd.sticky {
		delete(kd.sticky, key)
		kd.p.PutKey(key, kd.locked[key])
	}
}

// draw puts the diagram, with the keys down lit, at a position and width in
// pixels
func (kd *ebitenKeyboardDiagram) draw(dst *ebiten.Image, x, y float64, width int) {
	if width <= 0 {
		return
	}
	if kd.image == nil || kd.image.Bounds().Dx() != width {
		if kd.image != nil {
			kd.image.Deallocate()
			kd.pressed.Deallocate()
		}
		kd.image = ebiten.NewImageFromImage(kd.diagram.Render(width))
		kd.pressed = ebiten.NewImageFromImage(kd.diagram.RenderPressed(width))
	}

	var op ebiten.DrawImageOptions
	op.GeoM.Translate(x, y)
	dst.DrawImage(kd.image, &op)

	keys := kd.diagram.Keys()
	for i := range keys {
		if kd.ticks >= kd.shownUntil[keys[i].Key] && !kd.p.IsKeyDown(keys[i].Key) {
			continue
		}
		area := kd.diagram.KeyArea(&keys[i], width).Intersect(kd.pressed.Bounds())
		var kop ebiten.DrawImageOptions
		kop.GeoM.Translate(x+float64(area.Min.X), y+float64(area.Min.Y))
		dst.DrawImage(kd.pressed.SubImage(area).(*ebiten.Image), &kop)
	}
}
