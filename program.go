package izpet

import (
	"fmt"
	"strings"
)

/*
Loading of programs in .prg files, the load address on the first two bytes
followed by the data.

The program is put in memory once BASIC is ready, with READY. on the screen.
A BASIC program, loaded at the start of the BASIC text, gets the pointers to
the end of the program set as LOAD does, and RUN is typed.
*/

const basicStart = 0x0401

type pendingProgram struct {
	name string
	data []uint8
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

// LoadProgram loads a .prg file, when BASIC is ready
func (p *PET) LoadProgram(filename string) error {
	data, err := LoadResource(filename)
	if err != nil {
		return err
	}
	return p.loadProgramData(filename, data)
}

func (p *PET) loadProgramData(name string, data []uint8) error {
	if len(data) < 3 {
		return fmt.Errorf("%v is too short for a program", name)
	}
	address := uint16(data[0]) | uint16(data[1])<<8
	if int(address)+len(data)-2 > 0x10000 {
		return fmt.Errorf("%v does not fit in memory", name)
	}
	p.pendingPrograms = append(p.pendingPrograms, pendingProgram{name, data})
	return nil
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
		address := uint16(program.data[0]) | uint16(program.data[1])<<8
		for i, b := range program.data[2:] {
			p.mmu.Poke(address+uint16(i), b)
		}

		if address == basicStart {
			end := address + uint16(len(program.data)-2)
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
