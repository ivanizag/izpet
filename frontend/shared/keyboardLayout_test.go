package shared

import (
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/ivanizag/izpet"
)

func newTestPET(t *testing.T, model string) *izpet.PET {
	t.Helper()
	config := izpet.NewConfiguration()
	config.Model = model
	if err := config.Validate(); err != nil {
		t.Fatal(err)
	}
	p, err := izpet.NewPET(config)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// The keys on the diagram are the keys of the matrix, all of them except the
// ones of the 8032 editor with no key on the keyboard
func TestKeyboardDiagramsHaveAllTheKeys(t *testing.T) {
	for _, model := range []string{"2001", "3032", "8032"} {
		p := newTestPET(t, model)
		d := layoutFor(p)

		onDiagram := make(map[izpet.Key]bool)
		for i, k := range d.keys {
			if !p.HasKey(k.Key) {
				t.Errorf("%v: the key $%x is on the diagram and not on the keyboard", model, k.Key)
			}
			onDiagram[k.Key] = true

			if k.x < 0 || k.y < 0 || k.x+k.w > d.width || k.y+k.h > d.height {
				t.Errorf("%v: the key $%x is outside the diagram", model, k.Key)
			}
			for _, other := range d.keys[i+1:] {
				if k.x < other.x+other.w && other.x < k.x+k.w &&
					k.y < other.y+other.h && other.y < k.y+k.h {
					t.Errorf("%v: the keys $%x and $%x overlap", model, k.Key, other.Key)
				}
			}
		}
		for _, key := range p.Keys() {
			if !onDiagram[key] {
				t.Errorf("%v: the key $%x is not on the diagram", model, key)
			}
		}
	}
}

func TestKeyboardDiagramKeyAt(t *testing.T) {
	d := layoutFor(newTestPET(t, "8032"))
	unit := 1400 / d.width
	for _, k := range d.keys {
		got, ok := d.KeyAt((k.x+k.w/2)*unit, (k.y+k.h/2)*unit, 1400)
		if !ok || got.Key != k.Key {
			t.Errorf("the center of the key $%x gives the key $%x", k.Key, got.Key)
		}
	}
}

// Set IZPET_PREVIEW to a folder to get the diagrams as PNG images
func TestKeyboardDiagramPreview(t *testing.T) {
	dir := os.Getenv("IZPET_PREVIEW")
	if dir == "" {
		t.Skip("IZPET_PREVIEW not set")
	}
	for _, model := range []string{"2001", "3032", "8032"} {
		p := newTestPET(t, model)
		d, err := NewKeyboardDiagram(p)
		if err != nil {
			t.Fatal(err)
		}
		img := d.Render(1400)

		// Some keys down, as a frontend shows them
		pressed := d.RenderPressed(1400)
		for _, k := range d.Keys() {
			if k.Key == 'A' || k.Key == izpet.KeyReturn || k.Key == izpet.KeyShiftLeft || k.Key == '7' {
				area := d.KeyArea(&k, 1400)
				draw.Draw(img, area, pressed, area.Min, draw.Src)
			}
		}
		f, err := os.Create(filepath.Join(dir, "diagram_"+model+".png"))
		if err != nil {
			t.Fatal(err)
		}
		png.Encode(f, img)
		f.Close()
	}
}
