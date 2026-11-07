package izpet

import (
	"fmt"
	"os"
	"strings"

	"github.com/ivanizag/izpet/storage"
)

/*
The two cassette decks, the Commodore Datasette.

	        deck #1        deck #2
	read    PIA1 CA1       VIA CB1
	sense   PIA1 PA4       PIA1 PA5     low with a key pressed
	motor   PIA1 CB2       VIA PB4      low for on
	write   VIA PB3, the same for both

A tape is the list of the pulses on it, see storage/tape.go. With a tape in
the deck PLAY is pressed. The tape moves while the motor is on: the read line
goes low at the start of each pulse and high in the middle.

While the PET writes, the pulses it writes go on the tape, from where it is,
replacing what was there. A pulse written goes from a rising edge of the
write line to the next: the kernal times each half of a pulse on its own,
and the halves are paired this way when it reads them back. The tape is saved as a TAP file when the motor
stops after writing.

The emulation runs at full speed while a tape moves.
*/

const (
	// cassetteWriteActiveCycles is how long after an edge written the deck
	// is taken as recording, the longest pulse is about 700 cycles
	cassetteWriteActiveCycles = 20_000
)

type cassetteDeck struct {
	number   int
	filename string // Where to save the tape after recording
	pulses   []uint32
	inserted bool

	position    int    // The pulse under the head
	pulseCycles uint32 // Cycles into that pulse
	level       bool   // Of the read line

	motor            bool
	writeLevel       bool
	cyclesSinceWrite uint64 // Since the last rising edge written
	writing          bool
	modified         bool
}

func newCassetteDeck(number int) *cassetteDeck {
	return &cassetteDeck{number: number, level: true, cyclesSinceWrite: cassetteWriteActiveCycles}
}

// insert puts a tape in the deck. A file that does not exist is a blank
// tape, created as a TAP file when the PET writes on it.
func (c *cassetteDeck) insert(filename string) error {
	var pulses []uint32
	data, err := LoadResource(filename)
	if err == nil {
		pulses, err = storage.LoadTape(data)
		if err != nil {
			return err
		}
	} else if !os.IsNotExist(err) || !strings.HasSuffix(strings.ToLower(filename), ".tap") {
		return err
	}

	c.pulses = pulses
	c.filename = filename
	c.inserted = true
	c.position = 0
	c.pulseCycles = 0
	c.modified = false
	return nil
}

func (c *cassetteDeck) eject() {
	c.save()
	c.inserted = false
	c.pulses = nil
}

// isMoving returns true while the motor turns with a tape in the deck
func (c *cassetteDeck) isMoving() bool {
	return c.inserted && c.motor
}

// tick advances the tape, with the levels of the motor and write lines
func (c *cassetteDeck) tick(cycles uint64, motor bool, writeLevel bool) {
	if motor != c.motor {
		c.motor = motor
		if !motor {
			c.writing = false
			c.save()
		}
	}
	if !c.inserted || !motor {
		c.level = true
		return
	}

	c.cyclesSinceWrite += cycles
	if writeLevel != c.writeLevel {
		c.writeLevel = writeLevel
		if writeLevel {
			c.written()
		}
	}
	if c.cyclesSinceWrite >= cassetteWriteActiveCycles {
		c.writing = false
	}
	if c.writing {
		c.level = true
		return
	}

	// Reading
	c.pulseCycles += uint32(cycles)
	for c.position < len(c.pulses) && c.pulseCycles >= c.pulses[c.position] {
		c.pulseCycles -= c.pulses[c.position]
		c.position++
	}
	if c.position >= len(c.pulses) {
		c.level = true
		return
	}
	c.level = c.pulseCycles >= c.pulses[c.position]/2
}

// written records a rising edge of the write line, the end of a pulse
func (c *cassetteDeck) written() {
	if c.writing {
		pulse := uint32(c.cyclesSinceWrite)
		if c.position < len(c.pulses) {
			c.pulses[c.position] = pulse
		} else {
			c.pulses = append(c.pulses, pulse)
		}
		c.position++
		c.modified = true
	}
	c.writing = true
	c.cyclesSinceWrite = 0
	c.pulseCycles = 0
}

// save writes the tape to its file if the PET wrote on it
func (c *cassetteDeck) save() {
	if !c.modified || c.filename == "" {
		return
	}
	c.modified = false
	filename := c.filename
	if !strings.HasSuffix(strings.ToLower(filename), ".tap") {
		filename += ".tap"
	}
	err := os.WriteFile(filename, storage.MakeTap(c.pulses), 0644)
	if err != nil {
		fmt.Printf("Tape #%v: %v\n", c.number, err)
	}
}
