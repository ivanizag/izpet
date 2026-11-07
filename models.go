package izpet

import (
	"fmt"
	"sort"
	"strings"
)

/*
The PET models are combinations of a few choices: the ROM set, the video
board, the keyboard and the RAM. A model is a name for one combination.

See doc/plan.md for the family.
*/

type videoBoard int

const (
	// videoTTL is the video of the 2001 and the early 3000 and 4000
	// boards, made of discrete logic, with fixed 60 Hz timing
	videoTTL videoBoard = iota
	// videoCRTC40 is the 12" 4000 board, a 6545 CRTC for 40 columns
	videoCRTC40
	// videoCRTC80 is the 8000 board, the 6545 drives two characters per clock
	videoCRTC80
)

type keyboardLayout int

const (
	// keyboardGraphics is the matrix of the 2001 chiclet keyboard and of
	// the later graphics, or normal, keyboard
	keyboardGraphics keyboardLayout = iota
	// keyboardBusiness is the typewriter like keyboard
	keyboardBusiness
)

// romChip is a ROM image at an address
type romChip struct {
	address uint16
	file    string
}

type model struct {
	name        string
	description string
	basic       int // BASIC version, 1, 2 or 4
	roms        []romChip
	editor50Hz  string // The editor for 50 Hz, for the CRTC models only
	charGen     string
	video       videoBoard
	keyboard    keyboardLayout
	ramKB       int
}

const romPath = "<internal>/"

func romsBasic1() []romChip {
	return []romChip{
		{0xc000, romPath + "rom-2-c000.901439-09.bin"}, // The revised one
		{0xc800, romPath + "rom-1-c800.901439-05.bin"},
		{0xd000, romPath + "rom-1-d000.901439-02.bin"},
		{0xd800, romPath + "rom-1-d800.901439-06.bin"},
		{0xe000, romPath + "rom-1-e000.901439-03.bin"}, // Editor
		{0xf000, romPath + "rom-1-f000.901439-04.bin"},
		{0xf800, romPath + "rom-1-f800.901439-07.bin"},
	}
}

func romsBasic2(editor string) []romChip {
	return []romChip{
		{0xc000, romPath + "basic-2-c000.901465-01.bin"},
		{0xd000, romPath + "basic-2-d000.901465-02.bin"},
		{0xe000, romPath + editor},
		{0xf000, romPath + "kernal-2.901465-03.bin"},
	}
}

func romsBasic4(editor string) []romChip {
	return []romChip{
		{0xb000, romPath + "basic-4-b000.901465-23.bin"},
		{0xc000, romPath + "basic-4-c000.901465-20.bin"},
		{0xd000, romPath + "basic-4-d000.901465-21.bin"},
		{0xe000, romPath + editor},
		{0xf000, romPath + "kernal-4.901465-22.bin"},
	}
}

const (
	charGen1 = romPath + "characters-1.901447-08.bin"
	charGen2 = romPath + "characters-2.901447-10.bin"
)

func petModels() map[string]*model {
	models := []*model{
		{
			name:        "2001",
			description: "PET 2001, BASIC 1, chiclet keyboard",
			basic:       1,
			roms:        romsBasic1(),
			charGen:     charGen1,
			video:       videoTTL,
			keyboard:    keyboardGraphics,
			ramKB:       8,
		},
		{
			name:        "3032",
			description: "PET 2001-N or CBM 3032, BASIC 2, graphics keyboard",
			basic:       2,
			roms:        romsBasic2("edit-2-n.901447-24.bin"),
			charGen:     charGen2,
			video:       videoTTL,
			keyboard:    keyboardGraphics,
			ramKB:       32,
		},
		{
			name:        "3032b",
			description: "PET 2001-B or CBM 3032, BASIC 2, business keyboard",
			basic:       2,
			roms:        romsBasic2("edit-2-b.901474-01.bin"),
			charGen:     charGen2,
			video:       videoTTL,
			keyboard:    keyboardBusiness,
			ramKB:       32,
		},
		{
			name:        "4016",
			description: "CBM 4016, BASIC 4 on the 9\" board, graphics keyboard",
			basic:       4,
			roms:        romsBasic4("edit-4-n.901447-29.bin"),
			charGen:     charGen2,
			video:       videoTTL,
			keyboard:    keyboardGraphics,
			ramKB:       16,
		},
		{
			name:        "4032b",
			description: "CBM 4032, BASIC 4 on the 9\" board, business keyboard",
			basic:       4,
			roms:        romsBasic4("edit-4-b.901474-02.bin"),
			charGen:     charGen2,
			video:       videoTTL,
			keyboard:    keyboardBusiness,
			ramKB:       32,
		},
		{
			name:        "4032",
			description: "CBM 4032, BASIC 4 on the 12\" CRTC board, graphics keyboard",
			basic:       4,
			roms:        romsBasic4("edit-4-40-n-60Hz.901499-01.bin"),
			editor50Hz:  romPath + "edit-4-40-n-50Hz.901498-01.bin",
			charGen:     charGen2,
			video:       videoCRTC40,
			keyboard:    keyboardGraphics,
			ramKB:       32,
		},
		{
			name:        "8032",
			description: "CBM 8032, BASIC 4, 80 columns, business keyboard",
			basic:       4,
			roms:        romsBasic4("edit-4-80-b-60Hz.901474-03.bin"),
			editor50Hz:  romPath + "edit-4-80-b-50Hz.901474-04_alt.bin",
			charGen:     charGen2,
			video:       videoCRTC80,
			keyboard:    keyboardBusiness,
			ramKB:       32,
		},
	}

	m := make(map[string]*model)
	for _, model := range models {
		m[model.name] = model
	}
	return m
}

func getModel(name string) (*model, error) {
	models := petModels()
	m, ok := models[strings.ToLower(name)]
	if !ok {
		return nil, fmt.Errorf("unknown model '%v', the models are: %v",
			name, strings.Join(modelNames(), ", "))
	}
	return m, nil
}

func modelNames() []string {
	names := make([]string, 0)
	for name := range petModels() {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ModelsHelp returns a line per model with its description
func ModelsHelp() string {
	models := petModels()
	help := ""
	for _, name := range modelNames() {
		help += fmt.Sprintf("  %-6v %v\n", name, models[name].description)
	}
	return help
}
