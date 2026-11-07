package activities

import (
	"path/filepath"
	"testing"
)

// petModel is a model as the page presents it
type petModel struct {
	model  string
	label  string
	banner string
}

func pageModels() []petModel {
	return []petModel{
		{"2001", "PET 2001", "*** COMMODORE BASIC ***"},
		{"3032", "PET 2001-N / CBM 3032", "### COMMODORE BASIC ###"},
		{"3032b", "PET 2001-B / CBM 3032", "### COMMODORE BASIC ###"},
		{"4016", "CBM 4016", "*** COMMODORE BASIC 4.0 ***"},
		{"4032b", "CBM 4032, 9\"", "*** COMMODORE BASIC 4.0 ***"},
		{"4032", "CBM 4032, 12\"", "*** COMMODORE BASIC 4.0 ***"},
		{"8032", "CBM 8032", "*** COMMODORE BASIC 4.0 ***"},
	}
}

/*
modelsScreenshots is every model switched on: the start of the first one
recorded, from the screen the power on leaves to READY., and each one as it
waits for a command, with its keyboard
*/
func modelsScreenshots(t *testing.T) {
	const page = "models"
	album := NewAlbum(filepath.Join(activityImages, page))

	for i, m := range pageModels() {
		p := buildTestPET(t, testConfig(t, m.model))

		if i == 0 {
			// The first frames, and the start until READY. and a blink
			// of the cursor more
			starting := Record(p, m.label)
			for frame := 0; !hasText(p, "READY."); frame += 2 {
				if frame > 600 {
					t.Fatalf("the %v did not start", m.label)
				}
				starting.Run(2, 2)
			}
			starting.Run(60, 4)
			must(t, album.SaveRecording(starting, "starting", 300))
		}

		waitForText(t, p, m.banner, 600)
		waitForText(t, p, "READY.", 600)
		waitForCursor(t, p, "READY.")
		must(t, album.Machine(p, m.model, m.label))
	}
}
