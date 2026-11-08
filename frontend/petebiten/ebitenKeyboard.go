package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/ivanizag/izpet"
)

/*
The keyboard of the host on the PET.

The printable characters are typed by what they are, not by where the key
is: the layouts of the PET have the symbols in places a host keyboard does
not. A character typed on the host presses the PET key that types it, with
shift if needed. Shifted letters are typed with the PET shift, that makes
the graphics in the graphics mode and the uppercase in the lowercase mode.

The other keys are pressed by position for as long as they are held, with
the PET shift if a shift of the host is down when they are pressed.

Ctrl-V, Cmd-V and Shift-Insert paste the text of the clipboard.
*/
type ebitenKeyboard struct {
	p *izpet.PET
	g *game

	keys map[ebiten.Key]keyMapping
	down map[ebiten.Key]keyMapping
}

type keyMapping struct {
	key   izpet.Key
	shift bool // Always press shift with the key
}

func newEbitenKeyboard(p *izpet.PET, g *game) *ebitenKeyboard {
	var k ebitenKeyboard
	k.p = p
	k.g = g
	k.down = make(map[ebiten.Key]keyMapping)
	k.keys = map[ebiten.Key]keyMapping{
		ebiten.KeyEnter:      {izpet.KeyReturn, false},
		ebiten.KeyBackspace:  {izpet.KeyDel, false},
		ebiten.KeyDelete:     {izpet.KeyDel, false},
		ebiten.KeyInsert:     {izpet.KeyDel, true},
		ebiten.KeyHome:       {izpet.KeyHome, false},
		ebiten.KeyArrowDown:  {izpet.KeyCrsrDown, false},
		ebiten.KeyArrowUp:    {izpet.KeyCrsrDown, true},
		ebiten.KeyArrowRight: {izpet.KeyCrsrRight, false},
		ebiten.KeyArrowLeft:  {izpet.KeyCrsrRight, true},
		ebiten.KeyEscape:     {izpet.KeyStop, false},
		ebiten.KeyF3:         {izpet.KeyRvs, false},
	}

	if p.IsBusinessKeyboard() {
		k.keys[ebiten.KeyF9] = keyMapping{izpet.KeyEsc, false}
		k.keys[ebiten.KeyTab] = keyMapping{izpet.KeyTab, false}
		k.keys[ebiten.KeyNumpadDecimal] = keyMapping{izpet.KeyKeypadDot, false}
		for i := 0; i < 10; i++ {
			k.keys[ebiten.KeyNumpad0+ebiten.Key(i)] = keyMapping{izpet.KeyKeypad(i), false}
		}
	}
	return &k
}

func (k *ebitenKeyboard) update() {
	if !ebiten.IsFocused() {
		k.releaseAll()
		return
	}

	k.updateEmulatorKeys()

	control := ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyMeta)
	if !control {
		for _, ch := range ebiten.AppendInputChars(nil) {
			if k.isKeypadChar(ch) {
				// The business numeric pad goes by position
				continue
			}
			k.p.PutChar(ch)
		}
	}

	shift := ebiten.IsKeyPressed(ebiten.KeyShift)
	pasting := shift && inpututil.IsKeyJustPressed(ebiten.KeyInsert) ||
		control && inpututil.IsKeyJustPressed(ebiten.KeyV)
	if pasting {
		pasteClipboard(k.p)
	}

	for key, mapping := range k.keys {
		if inpututil.IsKeyJustPressed(key) && !(pasting && key == ebiten.KeyInsert) {
			mapping.shift = mapping.shift || shift
			k.press(key, mapping)
		}
		if inpututil.IsKeyJustReleased(key) {
			k.release(key)
		}
	}
}

func (k *ebitenKeyboard) isKeypadChar(ch rune) bool {
	if !k.p.IsBusinessKeyboard() || !(ch >= '0' && ch <= '9' || ch == '.') {
		return false
	}
	for i := 0; i < 10; i++ {
		if ebiten.IsKeyPressed(ebiten.KeyNumpad0 + ebiten.Key(i)) {
			return true
		}
	}
	return ebiten.IsKeyPressed(ebiten.KeyNumpadDecimal)
}

func (k *ebitenKeyboard) press(key ebiten.Key, mapping keyMapping) {
	k.down[key] = mapping
	if mapping.shift {
		k.p.PutKey(izpet.KeyShiftRight, true)
	}
	k.p.PutKey(mapping.key, true)
}

func (k *ebitenKeyboard) release(key ebiten.Key) {
	mapping, ok := k.down[key]
	if !ok {
		return
	}
	delete(k.down, key)
	k.p.PutKey(mapping.key, false)
	if mapping.shift && !k.isShiftStillNeeded() {
		k.p.PutKey(izpet.KeyShiftRight, false)
	}
}

func (k *ebitenKeyboard) isShiftStillNeeded() bool {
	for _, mapping := range k.down {
		if mapping.shift {
			return true
		}
	}
	return false
}

func (k *ebitenKeyboard) releaseAll() {
	for key := range k.down {
		k.release(key)
	}
}

// updateEmulatorKeys handles the keys that control the emulator
func (k *ebitenKeyboard) updateEmulatorKeys() {
	control := ebiten.IsKeyPressed(ebiten.KeyControl)
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyF1):
		k.g.showHelp = !k.g.showHelp
	case inpututil.IsKeyJustPressed(ebiten.KeyF5):
		if control {
			k.p.SendCommand(izpet.CommandShowSpeed)
		} else {
			k.p.SendCommand(izpet.CommandToggleSpeed)
		}
	case inpututil.IsKeyJustPressed(ebiten.KeyF4):
		k.p.SendCommand(izpet.CommandToggleCPUTrace)
	case inpututil.IsKeyJustPressed(ebiten.KeyF2):
		// While the help is shown, F2 alone resets. This is handy when the
		// window manager captures Ctrl-F2.
		if control || k.g.showHelp {
			k.p.SendCommand(izpet.CommandReset)
			k.g.showHelp = false
		}
	case inpututil.IsKeyJustPressed(ebiten.KeyF7):
		k.g.toggleKeyboard()
	case inpututil.IsKeyJustPressed(ebiten.KeyF12):
		saveScreenshot(k.p)
	case inpututil.IsKeyJustPressed(ebiten.KeyPause):
		k.p.SendCommand(izpet.CommandPauseUnpause)
	}
}
