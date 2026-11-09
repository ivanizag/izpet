package izpet

import (
	"bytes"
	"fmt"
	"strings"
)

/*
Loading of programs in .prg files, the load address on the first two bytes
followed by the data, and in .p00 files, the same after a header of 26
bytes.

The program is put in memory once BASIC is ready, with READY. on the screen.

A BASIC program goes at the start of the BASIC text, $0401, wherever it was
saved: a program of another Commodore computer, saved at $0801 by a C64 for
instance, is moved there. It is a BASIC program if it is saved at $0401, or
if its line links make a chain from where it was saved to the end of the
program. As LOAD does, the line links are made again for where it is, the
pointers to the end of the program are set, and RUN is typed.

Other programs, in machine language, are put where they were saved, and not
started.

A program has to fit in the RAM of the model, the part over $8000 is the
screen and the ROMs.
*/

const (
	basicStart = 0x0401

	p00Magic      = "C64File\x00"
	p00HeaderSize = 26
)

type pendingProgram struct {
	name    string
	address uint16
	data    []uint8 // Without the load address
	basic   bool
}

// Zero page pointer to the start of the BASIC variables, the end of the
// program. Followed by the pointers to the arrays and to the end of the
// arrays.
func (p *PET) basicVarTab() uint16 {
	if p.model.basic == 1 {
		return 0x7c
	}
	return 0x2a
}

// LoadProgram loads a .prg or .p00 file, when BASIC is ready
func (p *PET) LoadProgram(filename string) error {
	data, err := LoadResource(filename)
	if err != nil {
		return err
	}
	return p.loadProgramData(filename, data)
}

func (p *PET) loadProgramData(name string, data []uint8) error {
	if bytes.HasPrefix(data, []byte(p00Magic)) {
		if len(data) < p00HeaderSize {
			return fmt.Errorf("%v is too short for a .p00 file", name)
		}
		data = data[p00HeaderSize:]
	}
	if len(data) < 3 {
		return fmt.Errorf("%v is too short for a program", name)
	}

	program := pendingProgram{
		name:    name,
		address: uint16(data[0]) | uint16(data[1])<<8,
		data:    data[2:],
	}
	if program.address == basicStart || isBasicProgram(program.address, program.data) {
		program.basic = true
		program.address = basicStart
	}

	end := int(program.address) + len(program.data)
	if end > 0x10000 {
		return fmt.Errorf("%v does not fit in memory", name)
	}
	ram := len(p.mmu.ram)
	if int(program.address) < addressScreen && end > ram {
		return fmt.Errorf("%v needs memory up to $%04x, the PET has %v KB of RAM, up to $%04x",
			name, end-1, ram/1024, ram-1)
	}

	p.pendingPrograms = append(p.pendingPrograms, program)
	return nil
}

/*
isBasicProgram returns true if the data, saved at an address, is a BASIC
program: each line starts with the address of the next one and its number,
and ends with a zero just before the next one, up to a link of zero.
*/
func isBasicProgram(address uint16, data []uint8) bool {
	pos := 0
	for lines := 0; ; lines++ {
		if pos+1 >= len(data) {
			return false
		}
		link := int(data[pos]) | int(data[pos+1])<<8
		if link == 0 {
			return lines > 0
		}
		next := link - int(address)
		if next <= pos+4 || next > len(data) || data[next-1] != 0 {
			return false
		}
		pos = next
	}
}

// relink makes the line links of a BASIC program again for where it is
// loaded, as BASIC does after a LOAD: each line ends at its first zero
func relink(address uint16, data []uint8) {
	pos := 0
	for pos+1 < len(data) && (data[pos] != 0 || data[pos+1] != 0) {
		end := bytes.IndexByte(data[min(pos+4, len(data)):], 0)
		if end < 0 {
			return
		}
		next := pos + 4 + end + 1
		link := int(address) + next
		data[pos] = uint8(link)
		data[pos+1] = uint8(link >> 8)
		pos = next
	}
}

// isBasicReady returns true when BASIC waits for a command
func (p *PET) isBasicReady() bool {
	return !p.keyboard.isTyping() &&
		strings.Contains(strings.ToUpper(p.GetScreenText()), "READY.")
}

func (p *PET) loadPendingPrograms() {
	if len(p.pendingPrograms) == 0 || !p.isBasicReady() {
		return
	}

	for _, program := range p.pendingPrograms {
		data := program.data
		if program.basic {
			data = append([]uint8(nil), data...)
			relink(program.address, data)
		}
		for i, b := range data {
			p.mmu.Poke(program.address+uint16(i), b)
		}

		if program.basic {
			end := program.address + uint16(len(data))
			pointers := p.basicVarTab()
			for i := uint16(0); i < 3; i++ {
				p.mmu.Poke(pointers+2*i, uint8(end))
				p.mmu.Poke(pointers+2*i+1, uint8(end>>8))
			}
			p.keyboard.putText("RUN\r")
		}
	}
	p.pendingPrograms = nil
}
